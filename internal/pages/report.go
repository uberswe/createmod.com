package pages

import (
	"createmod/internal/mailer"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"

	"createmod/internal/server"
)

// ReportSubmitHandler handles POST /reports to create a simple report record.
func ReportSubmitHandler(mailService *mailer.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		targetType := e.Request.FormValue("target_type")
		targetID := e.Request.FormValue("target_id")
		reason := e.Request.FormValue("reason")
		returnTo := safeRedirectPath(e.Request.FormValue("return_to"), "/")
		if targetType == "" || targetID == "" || reason == "" {
			return e.String(http.StatusBadRequest, "missing required fields")
		}

		reporterID := authenticatedUserID(e)

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
			body := fmt.Sprintf("<p>Target: %s (%s)</p><p>Reason: %s</p>", targetID, targetType, reason)
			if reporterID != "" {
				body += fmt.Sprintf("<p>Reporter: %s</p>", reporterID)
			}
			msg := &mailer.Message{From: from, To: to, Subject: subject, HTML: body}
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
