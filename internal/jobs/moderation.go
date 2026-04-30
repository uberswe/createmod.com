package jobs

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/pages"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"os"

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

	// Run moderation if still in auto_review
	if schem.ModerationState == store.ModerationAutoReview && w.deps.Moderation != nil {
		var emailSubject, emailBodyText string

		// Step 1: Policy check (text + image if available)
		policyResult, policyErr := w.deps.Moderation.CheckSchematic(args.Title, args.Description, imageFullURL)
		if policyErr != nil {
			slog.Warn("moderation job: policy check unavailable", "error", policyErr, "schematic_id", args.SchematicID)
			emailSubject = fmt.Sprintf("Schematic Needs Review: %s", args.Title)
			emailBodyText = fmt.Sprintf("The schematic \"%s\" could not be auto-moderated (moderation unavailable). It requires manual review.", args.Title)
		} else if !policyResult.Approved {
			oldState := schem.ModerationState
			schem.ModerationState = store.ModerationFlagged
			schem.ModerationReason = policyResult.Reason
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to flag schematic", "error", updateErr, "schematic_id", args.SchematicID)
			} else {
				logStateChange(oldState, schem.ModerationState, "policy check failed: "+policyResult.Reason)
			}
			emailSubject = fmt.Sprintf("Schematic Blocked: %s", args.Title)
			emailBodyText = fmt.Sprintf("The schematic \"%s\" was blocked by automated moderation. Reason: %s", args.Title, policyResult.Reason)
		} else {
			// Step 2: Quality check
			qualityResult, qualityErr := w.deps.Moderation.CheckSchematicQuality(args.Title, args.Description)
			if qualityErr != nil {
				slog.Warn("moderation job: quality check unavailable", "error", qualityErr, "schematic_id", args.SchematicID)
				emailSubject = fmt.Sprintf("Schematic Needs Review: %s", args.Title)
				emailBodyText = fmt.Sprintf("The schematic \"%s\" could not be auto-moderated (quality check unavailable). It requires manual review.", args.Title)
			} else if !qualityResult.Approved {
				oldState := schem.ModerationState
				schem.ModerationState = store.ModerationFlagged
				schem.ModerationReason = qualityResult.Reason
				if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
					slog.Error("moderation job: failed to flag schematic", "error", updateErr, "schematic_id", args.SchematicID)
				} else {
					logStateChange(oldState, schem.ModerationState, "quality check failed: "+qualityResult.Reason)
				}
				emailSubject = fmt.Sprintf("Schematic Blocked: %s", args.Title)
				emailBodyText = fmt.Sprintf("The schematic \"%s\" was blocked by automated moderation. Reason: %s", args.Title, qualityResult.Reason)
			} else {
				// Both checks passed
				oldState := schem.ModerationState
				schem.ModerationState = store.ModerationPublished
				if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
					slog.Error("moderation job: failed to approve schematic", "error", updateErr, "schematic_id", args.SchematicID)
				} else {
					logStateChange(oldState, schem.ModerationState, "auto-approved: policy and quality checks passed")
				}
				emailSubject = fmt.Sprintf("Schematic Auto-Approved: %s", args.Title)
				emailBodyText = fmt.Sprintf("The schematic \"%s\" has been auto-approved and is now live on the site.", args.Title)
			}
		}

		// Send admin email
		if w.deps.Mail != nil && emailSubject != "" {
			schematicURL := fmt.Sprintf("%s/schematics/%s", baseURL, args.Slug)
			to := moderationAdminRecipients(w.deps.Store, w.deps.Mail)
			if len(to) > 0 {
				from := w.deps.Mail.DefaultFrom()
				body := mailer.SchematicEmailHTML(args.Title, imageFullURL, schematicURL, emailBodyText)
				msg := &mailer.Message{From: from, To: to, Subject: emailSubject, HTML: body}
				if sendErr := w.deps.Mail.Send(msg); sendErr != nil {
					slog.Error("moderation job: failed to send admin email", "error", sendErr)
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
		} else if !imgResult.Approved {
			slog.Warn("moderation job: featured image flagged, holding for review",
				"schematic_id", args.SchematicID, "reason", imgResult.Reason)
			oldState := schem.ModerationState
			schem.ModerationState = store.ModerationFlagged
			schem.ModerationReason = fmt.Sprintf("Featured image flagged by automated moderation: %s", imgResult.Reason)
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to hold schematic for review", "error", updateErr, "schematic_id", args.SchematicID)
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
		} else if !qualResult.Approved {
			slog.Warn("moderation job: featured image not a Minecraft build, holding for review",
				"schematic_id", args.SchematicID, "reason", qualResult.Reason)
			oldState := schem.ModerationState
			schem.ModerationState = store.ModerationFlagged
			schem.ModerationReason = fmt.Sprintf("Featured image is not a Minecraft build: %s", qualResult.Reason)
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to hold schematic for review", "error", updateErr, "schematic_id", args.SchematicID)
			} else {
				logStateChange(oldState, schem.ModerationState, "image quality check failed: "+qualResult.Reason)
			}
		}
	}

	// Run language detection and translation (regardless of moderation outcome)
	if w.deps.Translation != nil {
		w.deps.Translation.DetectAndTranslate(args.SchematicID)
	}

	if schem.ModerationState == store.ModerationPublished && w.deps.Cache != nil {
		pages.RefreshIndexCache(w.deps.Cache, w.deps.Store, []int{7})
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
