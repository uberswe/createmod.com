package pages

import (
	"createmod/internal/mailer"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"createmod/internal/server"
)

// allowedTargetTypes is the whitelist of target types that can be reported.
var allowedTargetTypes = map[string]bool{
	"schematic":  true,
	"comment":    true,
	"collection": true,
}

// ReportSubmitHandler handles POST /reports to create a simple report record.
// Public endpoint — authentication is optional but the reporter ID is recorded when available.
func ReportSubmitHandler(mailService *mailer.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}

		// Optional authentication — record reporter ID if logged in
		reporterID := authenticatedUserID(e)

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		targetType := e.Request.FormValue("target_type")
		targetID := e.Request.FormValue("target_id")
		reason := strings.TrimSpace(e.Request.FormValue("reason"))
		returnTo := safeRedirectPath(e.Request.FormValue("return_to"), "/")
		if targetType == "" || targetID == "" || reason == "" {
			return e.String(http.StatusBadRequest, "missing required fields")
		}

		// Validate target type against whitelist
		if !allowedTargetTypes[targetType] {
			return e.String(http.StatusBadRequest, "invalid target type")
		}

		// Validate target exists
		if !reportTargetExists(e, appStore, targetType, targetID) {
			return e.String(http.StatusBadRequest, "the reported content does not exist")
		}

		// Validate reason: require at least 10 characters of meaningful text
		if len(reason) < 10 {
			return e.String(http.StatusBadRequest, "please provide a more detailed reason (at least 10 characters)")
		}

		ctx := e.Request.Context()
		r := &store.Report{
			TargetType: targetType,
			TargetID:   targetID,
			Reason:     reason,
			Reporter:   reporterID,
		}
		if err := appStore.Reports.Create(ctx, r); err != nil {
			return e.String(http.StatusInternalServerError, "failed to save report")
		}

		// Best-effort email notification to all admins
		to := adminRecipients(appStore, mailService)
		if len(to) > 0 {
			from := mail.Address{Address: mailService.SenderAddress, Name: mailService.SenderName}
			subject := fmt.Sprintf("New Report: %s %s", targetType, targetID)
			bodyText := fmt.Sprintf("Target: %s (%s)\nReason: %s", targetID, targetType, reason)
			if reporterID != "" {
				bodyText += fmt.Sprintf("\nReporter: %s", reporterID)
			}
			htmlBody := mailer.EmailHTML(subject, "", "", "", bodyText)
			msg := &mailer.Message{From: from, To: to, Subject: subject, HTML: htmlBody}
			if err := mailService.Send(msg); err != nil {
				slog.Error("failed to send report email", "error", err)
			}
		}

		// HTMX-aware redirect
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, returnTo))
	}
}

// reportTargetExists checks whether the reported target actually exists in the database.
func reportTargetExists(e *server.RequestEvent, appStore *store.Store, targetType, targetID string) bool {
	ctx := e.Request.Context()
	switch targetType {
	case "schematic":
		s, err := appStore.Schematics.GetByID(ctx, targetID)
		return err == nil && s != nil
	case "comment":
		c, err := appStore.Comments.GetByID(ctx, targetID)
		return err == nil && c != nil
	case "collection":
		c, err := appStore.Collections.GetByID(ctx, targetID)
		return err == nil && c != nil
	default:
		return false
	}
}
