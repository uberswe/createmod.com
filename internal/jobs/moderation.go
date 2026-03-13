package jobs

import (
	"context"
	"createmod/internal/mailer"
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

	// Run moderation if not already resolved
	if !schem.Moderated && !schem.Blacklisted && w.deps.Moderation != nil {
		var emailSubject, emailBodyText string

		// Step 1: Policy check
		policyResult, policyErr := w.deps.Moderation.CheckSchematic(args.Title, args.Description, "")
		if policyErr != nil {
			slog.Warn("moderation job: policy check unavailable", "error", policyErr, "schematic_id", args.SchematicID)
			emailSubject = fmt.Sprintf("Schematic Needs Review: %s", args.Title)
			emailBodyText = fmt.Sprintf("The schematic \"%s\" could not be auto-moderated (moderation unavailable). It requires manual review.", args.Title)
		} else if !policyResult.Approved {
			schem.Blacklisted = true
			schem.ModerationReason = policyResult.Reason
			if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("moderation job: failed to blacklist schematic", "error", updateErr, "schematic_id", args.SchematicID)
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
				schem.Blacklisted = true
				schem.ModerationReason = qualityResult.Reason
				if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
					slog.Error("moderation job: failed to blacklist schematic", "error", updateErr, "schematic_id", args.SchematicID)
				}
				emailSubject = fmt.Sprintf("Schematic Blocked: %s", args.Title)
				emailBodyText = fmt.Sprintf("The schematic \"%s\" was blocked by automated moderation. Reason: %s", args.Title, qualityResult.Reason)
			} else {
				// Both checks passed
				schem.Moderated = true
				if updateErr := w.deps.Store.Schematics.Update(ctx, schem); updateErr != nil {
					slog.Error("moderation job: failed to approve schematic", "error", updateErr, "schematic_id", args.SchematicID)
				}
				emailSubject = fmt.Sprintf("Schematic Auto-Approved: %s", args.Title)
				emailBodyText = fmt.Sprintf("The schematic \"%s\" has been auto-approved and is now live on the site.", args.Title)
			}
		}

		// Send admin email
		if w.deps.Mail != nil && emailSubject != "" {
			baseURL := os.Getenv("BASE_URL")
			if baseURL == "" {
				baseURL = "https://createmod.com"
			}
			schematicURL := fmt.Sprintf("%s/schematics/%s", baseURL, args.Slug)
			var imageFullURL string
			if args.ImageURL != "" {
				imageFullURL = fmt.Sprintf("%s/api/files/schematics/%s/%s", baseURL, args.SchematicID, url.PathEscape(args.ImageURL))
			}

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

	// Run language detection and translation (regardless of moderation outcome)
	if w.deps.Translation != nil {
		w.deps.Translation.DetectAndTranslate(args.SchematicID)
	}

	slog.Info("async moderation complete", "schematic_id", args.SchematicID, "moderated", schem.Moderated, "blacklisted", schem.Blacklisted)
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
