package pages

import (
	"createmod/internal/mailer"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
)

// ContactSubmitHandler handles POST /api/contact to submit a contact form.
// Replaces PBs POST /api/collections/contact_form_submissions/records endpoint.
func ContactSubmitHandler(appStore *store.Store, mailService *mailer.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if err := e.Request.ParseMultipartForm(1 << 20); err != nil {
			if err2 := e.Request.ParseForm(); err2 != nil {
				return e.BadRequestError("invalid form data", nil)
			}
		}

		email := e.Request.FormValue("email")
		content := e.Request.FormValue("content")

		if email == "" || content == "" {
			return e.BadRequestError("email and content are required", nil)
		}

		// Save to database
		userID := authenticatedUserID(e)
		var authorID *string
		if userID != "" {
			authorID = &userID
		}
		ctx := e.Request.Context()
		if err := appStore.Contact.CreateSubmission(ctx, authorID, "Contact Form", content, email); err != nil {
			slog.Error("failed to save contact submission", "error", err)
			return e.InternalServerError("could not save submission", nil)
		}

		// Send email notification to all admins
		if mailService != nil {
			contactEmail := email
			contactContent := content
			go func() {
				to := adminRecipients(appStore, mailService)
				if len(to) == 0 {
					return
				}
				from := mail.Address{Address: mailService.SenderAddress, Name: mailService.SenderName}
				subject := "New CreateMod.com Contact Form Submission"
				body := fmt.Sprintf("<p>Email: %s</p><p>Content: %s</p>", contactEmail, contactContent)
				msg := &mailer.Message{From: from, To: to, Subject: subject, HTML: body}
				if err := mailService.Send(msg); err != nil {
					slog.Error("failed to send contact notification", "error", err)
				}
			}()
		}

		// HTMX-aware response
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Trigger", "contactSubmitted")
			return e.NoContent(http.StatusNoContent)
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
