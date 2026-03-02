package pages

import (
	"createmod/internal/store"
	"fmt"
	"net/http"
	"net/mail"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// ReportSubmitHandler handles POST /reports to create a simple report record in PB.
func ReportSubmitHandler(app *pocketbase.PocketBase, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		targetType := e.Request.FormValue("target_type")
		targetID := e.Request.FormValue("target_id")
		reason := e.Request.FormValue("reason")
		returnTo := e.Request.FormValue("return_to")
		if returnTo == "" {
			returnTo = "/"
		}
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

		// Best-effort email notification to superadmin
		super := os.Getenv("SUPERADMIN_EMAIL")
		if super == "" {
			super = app.Settings().Meta.SenderAddress
		}
		if super != "" {
			from := mail.Address{Address: app.Settings().Meta.SenderAddress, Name: app.Settings().Meta.SenderName}
			to := []mail.Address{{Address: super}}
			subject := fmt.Sprintf("New Report: %s %s", targetType, targetID)
			body := fmt.Sprintf("<p>Target: %s (%s)</p><p>Reason: %s</p>", targetID, targetType, reason)
			if reporterID != "" {
				body += fmt.Sprintf("<p>Reporter: %s</p>", reporterID)
			}
			msg := &mailer.Message{From: from, To: to, Subject: subject, HTML: body}
			if err := app.NewMailClient().Send(msg); err != nil {
				app.Logger().Error("failed to send report email", "error", err)
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
