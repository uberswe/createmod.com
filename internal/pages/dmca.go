package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/mailer"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
)

var dmcaTemplates = append([]string{
	"./template/dmca.html",
}, commonTemplates...)

type DMCAData struct {
	DefaultData
	Submitted bool // true after a successful no-JS form post redirect
}

// DMCAHandler renders the public DMCA takedown request page.
func DMCAHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := DMCAData{Submitted: e.Request.URL.Query().Get("submitted") == "1"}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "DMCA"))
		d.Title = i18n.T(d.Language, "DMCA Takedown Requests")
		d.Description = i18n.T(d.Language, "Submit a DMCA takedown request for content hosted on CreateMod.com.")
		d.Slug = "/dmca"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(dmcaTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// dmcaMaxFieldLen bounds every free-text field on the DMCA form.
const dmcaMaxFieldLen = 5000

// DMCASubmitHandler handles POST /api/dmca. Submissions are stored alongside
// contact form submissions and emailed to the admins.
func DMCASubmitHandler(appStore *store.Store, mailService *mailer.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if err := e.Request.ParseMultipartForm(1 << 20); err != nil {
			if err2 := e.Request.ParseForm(); err2 != nil {
				return e.BadRequestError("invalid form data", nil)
			}
		}

		field := func(name string) string {
			return strings.TrimSpace(e.Request.FormValue(name))
		}

		name := field("name")
		company := field("company")
		email := field("email")
		holder := field("copyright_holder")
		work := field("work")
		urls := field("urls")
		details := field("details")
		signature := field("signature")
		goodFaith := e.Request.FormValue("good_faith") != ""
		accuracy := e.Request.FormValue("accuracy") != ""

		if name == "" || email == "" || holder == "" || work == "" || urls == "" || signature == "" {
			return e.BadRequestError("please fill in all required fields", nil)
		}
		if !goodFaith || !accuracy {
			return e.BadRequestError("both statements must be confirmed", nil)
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return e.BadRequestError("invalid email address", nil)
		}
		for _, v := range []string{name, company, email, holder, work, urls, details, signature} {
			if len(v) > dmcaMaxFieldLen {
				return e.BadRequestError("field too long", nil)
			}
		}

		content := fmt.Sprintf(
			"Name: %s\nCompany: %s\nEmail: %s\nCopyright holder: %s\n\nCopyrighted work:\n%s\n\nAllegedly infringing URLs:\n%s\n\nAdditional details:\n%s\n\nGood-faith statement confirmed: %t\nAccuracy/authority statement confirmed (under penalty of perjury): %t\nSignature: %s",
			name, company, email, holder, work, urls, details, goodFaith, accuracy, signature,
		)

		userID := authenticatedUserID(e)
		var authorID *string
		if userID != "" {
			authorID = &userID
		}
		ctx := e.Request.Context()
		if err := appStore.Contact.CreateSubmission(ctx, authorID, "DMCA Request", content, email); err != nil {
			slog.Error("failed to save dmca submission", "error", err)
			return e.InternalServerError("could not save submission", nil)
		}

		if mailService != nil {
			notifyContent := content
			go func() {
				to := adminRecipients(appStore, mailService)
				if len(to) == 0 {
					return
				}
				from := mail.Address{Address: mailService.SenderAddress, Name: mailService.SenderName}
				subject := "New DMCA Request"
				htmlBody := mailer.EmailHTML(subject, "", "", "", notifyContent)
				msg := &mailer.Message{From: from, To: to, Subject: subject, HTML: htmlBody}
				if err := mailService.Send(msg); err != nil {
					slog.Error("failed to send dmca notification", "error", err)
				}
			}()
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/dmca?submitted=1"))
			return e.NoContent(http.StatusNoContent)
		}
		if strings.Contains(e.Request.Header.Get("Accept"), "text/html") {
			// Plain (no-JS) form post — land back on the page with a banner.
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/dmca?submitted=1"))
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
