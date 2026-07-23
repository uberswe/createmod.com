package jobs

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/pages"
	"createmod/internal/store"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"os"
	"time"

	"github.com/riverqueue/river"
)

// ModerationArgs are the arguments for the async schematic moderation job.
type ModerationArgs struct {
	SchematicID string `json:"schematic_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	Slug        string `json:"slug"`
}

func (ModerationArgs) Kind() string { return "schematic_moderation" }

// ModerationWorker runs OpenAI moderation checks asynchronously and sends admin email.
type ModerationWorker struct {
	river.WorkerDefaults[ModerationArgs]
	deps Deps
}

func (w *ModerationWorker) Work(ctx context.Context, job *river.Job[ModerationArgs]) error {
	args := job.Args
	slog.Info("running async moderation", "schematic_id", args.SchematicID, "title", args.Title)

	if w.deps.Store == nil {
		slog.Warn("moderation job skipped: missing store", "schematic_id", args.SchematicID)
		return nil
	}

	// Load schematic to ensure it still exists
	schem, err := w.deps.Store.Schematics.GetByID(ctx, args.SchematicID)
	if err != nil || schem == nil {
		slog.Warn("moderation job: schematic not found, skipping", "schematic_id", args.SchematicID, "error", err)
		return nil
	}

	// Build the full public URL for the featured image (used by moderation and email).
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://createmod.com"
	}
	var imageFullURL string
	if args.ImageURL != "" {
		imageFullURL = fmt.Sprintf("%s/api/files/schematics/%s/%s", baseURL, args.SchematicID, url.PathEscape(args.ImageURL))
	}

	logStateChange := func(oldState, newState, reason string) {
		if w.deps.Store.ModerationLog != nil {
			_ = w.deps.Store.ModerationLog.Create(ctx, &store.ModerationLogEntry{
				SchematicID: args.SchematicID,
				ActorType:   "system",
				Action:      "state_change",
				OldState:    oldState,
				NewState:    newState,
				Reason:      reason,
			})
		}
	}

	// Failures below are collected and returned so River retries the job.
	// The checks are idempotent: the policy/quality section only runs while
	// the schematic is still in auto_review, and the image checks only flag.
	// Retries use the slow schedule in NextRetry — these failures usually
	// mean OpenAI or the database is unavailable, and quick retries would
	// burn attempts against the same outage. (Before 2026-07-23 these were
	// logged and swallowed, which hid a 15-hour moderation outage behind
	// "completed" jobs.)
	var errs []error

	// Run moderation if still in auto_review. Outcomes are reported to admins
	// via the twice-daily moderation summary email, not per-event emails:
	// approvals appear in the auto-approved section, flagged and
	// still-in-auto-review schematics in the pending list.
	if schem.ModerationState == store.ModerationAutoReview && w.deps.Moderation != nil {
		// Step 1: Policy check (text + image if available)
		policyResult, policyErr := w.deps.Moderation.CheckSchematic(args.Title, args.Description, imageFullURL)
		if policyErr != nil {
			slog.Warn("moderation job: policy check unavailable", "error", policyErr, "schematic_id", args.SchematicID)
			errs = append(errs, fmt.Errorf("policy check: %w", policyErr))
		} else if !policyResult.Approved {
			oldState := schem.ModerationState
			schem.ModerationState = store.ModerationFlagged
			schem.ModerationReason = policyResult.Reason
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to flag schematic", "error", updateErr, "schematic_id", args.SchematicID)
				errs = append(errs, fmt.Errorf("flagging schematic: %w", updateErr))
			} else {
				logStateChange(oldState, schem.ModerationState, "policy check failed: "+policyResult.Reason)
			}
		} else {
			// Step 2: Quality check
			qualityResult, qualityErr := w.deps.Moderation.CheckSchematicQuality(args.Title, args.Description)
			if qualityErr != nil {
				slog.Warn("moderation job: quality check unavailable", "error", qualityErr, "schematic_id", args.SchematicID)
				errs = append(errs, fmt.Errorf("quality check: %w", qualityErr))
			} else if !qualityResult.Approved {
				oldState := schem.ModerationState
				schem.ModerationState = store.ModerationFlagged
				schem.ModerationReason = qualityResult.Reason
				if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
					slog.Error("moderation job: failed to flag schematic", "error", updateErr, "schematic_id", args.SchematicID)
					errs = append(errs, fmt.Errorf("flagging schematic: %w", updateErr))
				} else {
					logStateChange(oldState, schem.ModerationState, "quality check failed: "+qualityResult.Reason)
				}
			} else {
				// Both checks passed
				oldState := schem.ModerationState
				schem.ModerationState = store.ModerationPublished
				if schem.ScheduledAt != nil && schem.ScheduledAt.After(time.Now()) {
					schem.CreatedOverride = schem.ScheduledAt
				}
				if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
					slog.Error("moderation job: failed to approve schematic", "error", updateErr, "schematic_id", args.SchematicID)
					errs = append(errs, fmt.Errorf("approving schematic: %w", updateErr))
				} else {
					logStateChange(oldState, schem.ModerationState, "auto-approved: policy and quality checks passed")
				}
			}
		}
	}

	// Always run image safety check (even for trusted/pre-approved users).
	// This catches policy-violating images that bypassed moderation via auto-approval.
	if w.deps.Moderation != nil && imageFullURL != "" && schem.ModerationState != store.ModerationDeleted {
		imgResult, imgErr := w.deps.Moderation.CheckImage(imageFullURL)
		if imgErr != nil {
			slog.Warn("moderation job: image safety check unavailable", "error", imgErr, "schematic_id", args.SchematicID)
			errs = append(errs, fmt.Errorf("image safety check: %w", imgErr))
		} else if !imgResult.Approved {
			slog.Warn("moderation job: featured image flagged, holding for review",
				"schematic_id", args.SchematicID, "reason", imgResult.Reason)
			oldState := schem.ModerationState
			schem.ModerationState = store.ModerationFlagged
			schem.ModerationReason = fmt.Sprintf("Featured image flagged by automated moderation: %s", imgResult.Reason)
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to hold schematic for review", "error", updateErr, "schematic_id", args.SchematicID)
				errs = append(errs, fmt.Errorf("holding schematic (image safety): %w", updateErr))
			} else {
				logStateChange(oldState, schem.ModerationState, "image safety check failed: "+imgResult.Reason)
			}
		}
	}

	// Always run image quality check (even for trusted/pre-approved users).
	// This verifies the featured image depicts an actual Minecraft build, catching
	// off-topic uploads like anime characters or unrelated photos.
	if w.deps.Moderation != nil && imageFullURL != "" && schem.ModerationState != store.ModerationDeleted {
		qualResult, qualErr := w.deps.Moderation.CheckImageQuality(imageFullURL)
		if qualErr != nil {
			slog.Warn("moderation job: image quality check unavailable", "error", qualErr, "schematic_id", args.SchematicID)
			errs = append(errs, fmt.Errorf("image quality check: %w", qualErr))
		} else if !qualResult.Approved {
			slog.Warn("moderation job: featured image not a Minecraft build, holding for review",
				"schematic_id", args.SchematicID, "reason", qualResult.Reason)
			oldState := schem.ModerationState
			schem.ModerationState = store.ModerationFlagged
			schem.ModerationReason = fmt.Sprintf("Featured image is not a Minecraft build: %s", qualResult.Reason)
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to hold schematic for review", "error", updateErr, "schematic_id", args.SchematicID)
				errs = append(errs, fmt.Errorf("holding schematic (image quality): %w", updateErr))
			} else {
				logStateChange(oldState, schem.ModerationState, "image quality check failed: "+qualResult.Reason)
			}
		}
	}

	// Run language detection and translation (regardless of moderation outcome)
	if w.deps.Translation != nil {
		w.deps.Translation.DetectAndTranslate(args.SchematicID)
	}

	if schem.ModerationState == store.ModerationPublished {
		if w.deps.Cache != nil {
			pages.RefreshIndexCache(w.deps.Cache, w.deps.Store, []int{7})
		}
		// Immediately index the newly published schematic in Meilisearch.
		if w.deps.MeiliClient != nil {
			upsertSchematicToMeili(ctx, w.deps, args.SchematicID)
		}
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		slog.Warn("async moderation incomplete, will retry",
			"schematic_id", args.SchematicID, "attempt", job.Attempt, "error", err)
		return err
	}

	slog.Info("async moderation complete", "schematic_id", args.SchematicID, "moderation_state", schem.ModerationState)
	return nil
}

// NextRetry applies the slow retry schedule (30m doubling to a 24h ceiling):
// moderation failures usually mean OpenAI or the database is down.
func (w *ModerationWorker) NextRetry(job *river.Job[ModerationArgs]) time.Time {
	return slowRetryAt(job.Attempt, time.Now())
}

// moderationAdminRecipients returns mail.Address entries for admin users.
// Duplicated from internal/pages to avoid import cycle.
func moderationAdminRecipients(appStore *store.Store, mailService *mailer.Service) []mail.Address {
	if appStore != nil {
		emails, err := appStore.Users.ListAdminEmails(context.Background())
		if err == nil && len(emails) > 0 {
			addrs := make([]mail.Address, len(emails))
			for i, e := range emails {
				addrs[i] = mail.Address{Address: e}
			}
			return addrs
		}
	}
	from := mailService.DefaultFrom()
	if from.Address != "" {
		return []mail.Address{from}
	}
	return nil
}
