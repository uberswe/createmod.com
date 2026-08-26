package jobs

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/pages"
	"createmod/internal/ratelimit"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"os"
	"strings"
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

	// Run moderation if still in auto_review. Outcomes are reported to admins
	// via the twice-daily moderation summary email, not per-event emails:
	// approvals appear in the auto-approved section, flagged and
	// still-in-auto-review schematics in the pending list.
	//
	// A completed run MUST leave a terminal state: auto_review is invisible to
	// everyone and absent from the moderator queue, so a schematic stranded
	// there (nil moderation service, or a check that keeps erroring) is silently
	// lost. Every branch below therefore resolves the schematic, and transient
	// check failures retry a few times before failing open. (#1646)
	if schem.ModerationState == store.ModerationAutoReview {
		// publish moves the schematic to its live state (honouring a scheduled
		// publish time) and logs why.
		publish := func(reason string) {
			oldState := schem.ModerationState
			schem.ModerationState = store.ModerationPublished
			if schem.ScheduledAt != nil && schem.ScheduledAt.After(time.Now()) {
				schem.CreatedOverride = schem.ScheduledAt
			}
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to publish schematic", "error", updateErr, "schematic_id", args.SchematicID)
			} else {
				logStateChange(oldState, schem.ModerationState, reason)
			}
		}

		if w.deps.Moderation == nil {
			// No moderation service configured: fail open so nothing is stranded.
			publish("auto-published: moderation service unavailable")
		} else if policyResult, policyErr := w.deps.Moderation.CheckSchematic(args.Title, args.Description, imageFullURL); policyErr != nil {
			// Transient failure: retry a few times (River backoff), then fail open
			// so a persistent outage can't strand the schematic. The periodic
			// auto_review backfill is the long-stop if a run is ever discarded.
			const maxPolicyAttempts = 3
			if job.Attempt < maxPolicyAttempts {
				return fmt.Errorf("moderation policy check for %s failed on attempt %d, retrying: %w", args.SchematicID, job.Attempt, policyErr)
			}
			slog.Error("moderation job: policy check still failing, publishing to avoid stranding", "error", policyErr, "schematic_id", args.SchematicID, "attempt", job.Attempt)
			publish("auto-published: policy check unavailable after retries")
		} else if !policyResult.Approved {
			oldState := schem.ModerationState
			schem.ModerationState = store.ModerationFlagged
			schem.ModerationReason = policyResult.Reason
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to flag schematic", "error", updateErr, "schematic_id", args.SchematicID)
			} else {
				_ = w.deps.Store.Schematics.SetModerationReviewedBy(ctx, args.SchematicID, store.ReviewedBySystem)
				logStateChange(oldState, schem.ModerationState, "policy check failed: "+policyResult.Reason)
			}
		} else if qualityResult, qualityErr := w.deps.Moderation.CheckSchematicQuality(args.Title, args.Description); qualityErr != nil {
			// Quality only gates Latest/search visibility, not policy, so an
			// inability to check it publishes rather than strands the schematic.
			slog.Warn("moderation job: quality check unavailable, publishing", "error", qualityErr, "schematic_id", args.SchematicID)
			publish("auto-published: quality check unavailable")
		} else if !qualityResult.Approved {
			// Quality failure is NOT a policy violation: publish-first with
			// limits instead of holding for manual review. The schematic is
			// reachable by its link but excluded from Latest/search until the
			// author improves the description; a checklist item tells them how,
			// and resolving it auto-promotes (no moderator needed). (#1646)
			if pages.EnterPublishedLimited(ctx, w.deps.Store, schem, qualityResult.Reason) {
				pages.SendSchematicActionNeededEmail(ctx, w.deps.Mail, w.deps.Store, args.SchematicID)
			}
		} else {
			publish("auto-approved: policy and quality checks passed")
		}
	}

	// Featured-image safety + quality checks run even for trusted/pre-approved
	// users, catching policy-violating or off-topic images that bypassed
	// moderation via auto-approval. A failure no longer holds the whole
	// schematic: the individual featured image is put on hold (hidden from
	// visitors, placeholder for the owner) and featured falls back to a visible
	// gallery image, so the schematic stays published. (#1646)
	if w.deps.Moderation != nil && imageFullURL != "" && args.ImageURL != "" && schem.ModerationState != store.ModerationDeleted {
		var holdReasons []string
		if imgResult, imgErr := w.deps.Moderation.CheckImage(imageFullURL); imgErr != nil {
			slog.Warn("moderation job: image safety check unavailable", "error", imgErr, "schematic_id", args.SchematicID)
		} else if !imgResult.Approved {
			holdReasons = append(holdReasons, "featured image flagged by automated moderation: "+imgResult.Reason)
		}
		if qualResult, qualErr := w.deps.Moderation.CheckImageQuality(imageFullURL); qualErr != nil {
			slog.Warn("moderation job: image quality check unavailable", "error", qualErr, "schematic_id", args.SchematicID)
		} else if !qualResult.Approved {
			holdReasons = append(holdReasons, "featured image is not a Minecraft build: "+qualResult.Reason)
		}
		if len(holdReasons) > 0 {
			slog.Warn("moderation job: holding featured image", "schematic_id", args.SchematicID, "reasons", holdReasons)
			pages.HoldSchematicImages(ctx, w.deps.Mail, w.deps.Store, args.SchematicID, []string{args.ImageURL}, strings.Join(holdReasons, "; "))
		}
	}

	// Language detection + translation is a paid OpenAI call, so it draws on the
	// author's hourly paid-AI budget. Over budget we DEFER: the bounded, deduped
	// translation backfill reconciles the missing translation on a later cycle,
	// so rapid uploads/edits can't run up translation cost. (#1646)
	if w.deps.Translation != nil {
		if ratelimit.AllowPaidAI(ctx, w.deps.RateLimiter, schem.AuthorID) {
			w.deps.Translation.DetectAndTranslate(args.SchematicID)
		} else {
			slog.Info("moderation: over paid-AI budget, deferring translation to backfill",
				"schematic_id", args.SchematicID, "author_id", schem.AuthorID)
		}
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

	slog.Info("async moderation complete", "schematic_id", args.SchematicID, "moderation_state", schem.ModerationState)
	return nil
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
