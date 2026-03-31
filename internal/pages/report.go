package pages

import (
	"createmod/internal/mailer"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"regexp"

	"createmod/internal/server"
)

// validReportTargetTypes is the set of allowed target_type values.
var validReportTargetTypes = map[string]bool{
	"schematic":  true,
	"comment":    true,
	"collection": true,
}

// reportIDPattern matches valid IDs: 15-char alphanumeric (legacy) or UUID.
var reportIDPattern = regexp.MustCompile(`^[a-z0-9]{15}$|^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ReportSubmitHandler handles POST /reports to create a simple report record.
func ReportSubmitHandler(mailService *mailer.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := e.Request.ParseMultipartForm(32 << 10); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		targetType := e.Request.FormValue("target_type")
		targetID := e.Request.FormValue("target_id")
		reason := e.Request.FormValue("reason")
		returnTo := safeRedirectPath(e.Request.FormValue("return_to"), "/")
		if targetType == "" || targetID == "" || reason == "" {
			return e.String(http.StatusBadRequest, "missing required fields")
		}
		if !validReportTargetTypes[targetType] {
			return e.String(http.StatusBadRequest, "invalid target type")
		}
		if !reportIDPattern.MatchString(targetID) {
			return e.String(http.StatusBadRequest, "invalid target id")
		}
		if len(reason) > 2000 {
			return e.String(http.StatusBadRequest, "reason too long")
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
