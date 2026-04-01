package pages

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strings"

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
			base := reportBaseURL()
			targetPath, targetLabel := resolveReportTarget(appStore, targetType, targetID)
			targetURL := ""
			if targetPath != "" {
				targetURL = base + targetPath
			} else {
				targetURL = base + "/admin/reports"
			}
			imageURL := reportTargetImage(ctx, appStore, targetType, targetID, base)
			reporterLabel := resolveReportUser(ctx, appStore, reporterID)

			from := mail.Address{Address: mailService.SenderAddress, Name: mailService.SenderName}
			subject := fmt.Sprintf("New Report: %s — %s", targetType, targetLabel)
			bodyText := fmt.Sprintf("Target: %s (%s)\nReason: %s\nReporter: %s", targetLabel, targetType, reason, reporterLabel)
			buttonLabel := "View " + strings.ToUpper(targetType[:1]) + targetType[1:]
			htmlBody := mailer.EmailHTML(subject, imageURL, targetURL, buttonLabel, bodyText)
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

// reportBaseURL returns the site base URL from env or the default.
func reportBaseURL() string {
	if u := os.Getenv("BASE_URL"); u != "" {
		return u
	}
	return "https://createmod.com"
}

// reportTargetImage returns a featured image URL for schematic reports, or empty string.
func reportTargetImage(ctx context.Context, appStore *store.Store, targetType, targetID, baseURL string) string {
	if targetType != "schematic" {
		return ""
	}
	s, err := appStore.Schematics.GetByID(ctx, targetID)
	if err == nil && s != nil && s.FeaturedImage != "" {
		return baseURL + "/api/files/schematics/" + s.ID + "/" + url.PathEscape(s.FeaturedImage)
	}
	return ""
}

// resolveReportUser returns a display string for the reporter.
func resolveReportUser(ctx context.Context, appStore *store.Store, userID string) string {
	if userID == "" {
		return "anonymous"
	}
	u, err := appStore.Users.GetUserByID(ctx, userID)
	if err == nil && u != nil {
		return u.Username + " (" + userID + ")"
	}
	return userID
}
