package pages

import (
	"context"
	"log/slog"
	"net/mail"
	"os"

	"createmod/internal/mailer"
	"createmod/internal/store"
)

// moderationEmailTarget resolves the author's email and the schematic's public
// URL for a moderation notification. Returns ok=false when the schematic or
// author email is missing (nothing to send). (#1646)
func moderationEmailTarget(ctx context.Context, appStore *store.Store, schematicID string) (schem *store.Schematic, email, url string, ok bool) {
	if appStore == nil {
		return nil, "", "", false
	}
	schem, err := appStore.Schematics.GetByID(ctx, schematicID)
	if err != nil || schem == nil || schem.AuthorID == "" {
		return nil, "", "", false
	}
	author, err := appStore.Users.GetUserByID(ctx, schem.AuthorID)
	if err != nil || author == nil || author.Email == "" {
		return nil, "", "", false
	}
	base := os.Getenv("BASE_URL")
	if base == "" {
		base = "https://createmod.com"
	}
	return schem, author.Email, base + "/schematics/" + schem.Name, true
}

func sendModerationEmail(mailService *mailer.Service, to, subject, htmlBody string) {
	if mailService == nil || to == "" {
		return
	}
	msg := &mailer.Message{
		From:    mailService.DefaultFrom(),
		To:      []mail.Address{{Address: to}},
		Subject: subject,
		HTML:    htmlBody,
	}
	if err := mailService.Send(msg); err != nil {
		slog.Error("moderation email: send failed", "to", to, "subject", subject, "error", err)
	}
}

// SendSchematicLiveEmail tells the author their schematic is fully live.
func SendSchematicLiveEmail(ctx context.Context, mailService *mailer.Service, appStore *store.Store, schematicID string) {
	schem, email, url, ok := moderationEmailTarget(ctx, appStore, schematicID)
	if !ok {
		return
	}
	sendModerationEmail(mailService, email, "Your schematic "+schem.Title+" is live",
		mailer.SchematicLiveEmail(schem.Title, url))
}

// SendSchematicActionNeededEmail tells the author their schematic published with
// limits, listing the open checklist items to resolve.
func SendSchematicActionNeededEmail(ctx context.Context, mailService *mailer.Service, appStore *store.Store, schematicID string) {
	schem, email, url, ok := moderationEmailTarget(ctx, appStore, schematicID)
	if !ok {
		return
	}
	var items []string
	if appStore.ModerationChecklist != nil {
		open, _ := appStore.ModerationChecklist.ListOpenBySchematic(ctx, schematicID)
		for _, it := range open {
			if it.Note != "" {
				items = append(items, it.Note)
			}
		}
	}
	sendModerationEmail(mailService, email, "Action needed: unlock full visibility for "+schem.Title,
		mailer.SchematicActionNeededEmail(schem.Title, url, items))
}

// SendSchematicImageReviewEmail tells the author an image is under review.
func SendSchematicImageReviewEmail(ctx context.Context, mailService *mailer.Service, appStore *store.Store, schematicID string) {
	schem, email, url, ok := moderationEmailTarget(ctx, appStore, schematicID)
	if !ok {
		return
	}
	sendModerationEmail(mailService, email, "One image on "+schem.Title+" is being reviewed",
		mailer.SchematicImageReviewEmail(schem.Title, url, moderationSLAHours()))
}

// SendSchematicNotPublishedEmail tells the author their schematic was rejected.
func SendSchematicNotPublishedEmail(ctx context.Context, mailService *mailer.Service, appStore *store.Store, schematicID string, fixable bool) {
	schem, email, url, ok := moderationEmailTarget(ctx, appStore, schematicID)
	if !ok {
		return
	}
	sendModerationEmail(mailService, email, schem.Title+" was not published",
		mailer.SchematicNotPublishedEmail(schem.Title, url, schem.ModerationReason, fixable))
}
