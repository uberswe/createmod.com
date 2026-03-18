package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"createmod/internal/webhook"
	"net/http"
	"net/url"
	"strings"
)

var userWebhooksTemplates = append([]string{
	"./template/user-webhooks.html",
}, commonTemplates...)

// UserWebhookData holds data for the webhook settings page.
type UserWebhookData struct {
	DefaultData
	HasWebhook          bool
	WebhookMasked       string
	WebhookActive       bool
	ConsecutiveFailures int
	LastFailureMessage  string
	SuccessMessage      string
	ErrorMessage        string
}

// webhookRedirect performs an HTMX-aware redirect to /settings/webhooks with
// an optional query parameter.
func webhookRedirect(e *server.RequestEvent, paramKey, paramValue string) error {
	dest := "/settings/webhooks"
	if paramKey != "" {
		dest += "?" + paramKey + "=" + url.QueryEscape(paramValue)
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
}

// UserWebhooksHandler handles GET /settings/webhooks
func UserWebhooksHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, webhookSecret string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)

		d := UserWebhookData{}
		d.Populate(e)
		d.HideOutstream = true
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Webhooks"))
		d.Title = i18n.T(d.Language, "Webhooks")
		d.Slug = "/settings/webhooks"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		// Check for flash messages from query params
		if msg := e.Request.URL.Query().Get("success"); msg != "" {
			d.SuccessMessage = msg
		}
		if msg := e.Request.URL.Query().Get("error"); msg != "" {
			d.ErrorMessage = msg
		}

		// Load user's webhook
		ctx := e.Request.Context()
		wh, err := appStore.Webhooks.GetByUserID(ctx, userID)
		if err == nil && wh != nil {
			d.HasWebhook = true
			d.WebhookActive = wh.Active
			d.ConsecutiveFailures = wh.ConsecutiveFailures
			d.LastFailureMessage = wh.LastFailureMessage

			// Decrypt and mask the URL
			plainURL, decErr := webhook.Decrypt(wh.WebhookURLEncrypted, webhookSecret)
			if decErr == nil && len(plainURL) > 10 {
				d.WebhookMasked = "••••••••••" + plainURL[len(plainURL)-10:]
			} else if decErr == nil {
				d.WebhookMasked = "••••••••••"
			} else {
				d.WebhookMasked = "(unable to read)"
			}
		}

		html, err := registry.LoadFiles(userWebhooksTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// UserWebhookSaveHandler handles POST /settings/webhooks
func UserWebhookSaveHandler(cacheService *cache.Service, appStore *store.Store, webhookSecret string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}

		userID := authenticatedUserID(e)
		rawURL := strings.TrimSpace(e.Request.FormValue("webhook_url"))

		// Validate
		if err := webhook.ValidateDiscordWebhookURL(rawURL); err != nil {
			return webhookRedirect(e, "error", err.Error())
		}

		// Encrypt
		encrypted, err := webhook.Encrypt(rawURL, webhookSecret)
		if err != nil {
			return webhookRedirect(e, "error", "Failed to encrypt webhook URL")
		}

		// Atomic upsert — no race condition between check and write
		ctx := e.Request.Context()
		if err := appStore.Webhooks.Upsert(ctx, userID, encrypted); err != nil {
			return webhookRedirect(e, "error", "Failed to save webhook")
		}

		return webhookRedirect(e, "success", "Webhook saved successfully")
	}
}

// UserWebhookDeleteHandler handles POST /settings/webhooks/delete
func UserWebhookDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()
		if err := appStore.Webhooks.Delete(ctx, userID); err != nil {
			return webhookRedirect(e, "error", "Failed to delete webhook")
		}

		return webhookRedirect(e, "success", "Webhook removed")
	}
}
