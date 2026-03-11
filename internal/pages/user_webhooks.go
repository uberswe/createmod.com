package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"createmod/internal/webhook"
	"net/http"
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

// UserWebhooksHandler handles GET /settings/webhooks
func UserWebhooksHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, webhookSecret string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)

		d := UserWebhookData{}
		d.Populate(e)
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
			dest := "/settings/webhooks?error=" + err.Error()
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
		}

		// Encrypt
		encrypted, err := webhook.Encrypt(rawURL, webhookSecret)
		if err != nil {
			dest := "/settings/webhooks?error=Failed to encrypt webhook URL"
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
		}

		// Upsert: try update first, if no rows exist, create
		ctx := context.Background()
		_, getErr := appStore.Webhooks.GetByUserID(ctx, userID)
		if getErr != nil {
			// No existing webhook — create
			if err := appStore.Webhooks.Create(ctx, userID, encrypted); err != nil {
				dest := "/settings/webhooks?error=Failed to save webhook"
				if e.Request.Header.Get("HX-Request") != "" {
					e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
					return e.HTML(http.StatusNoContent, "")
				}
				return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
			}
		} else {
			// Existing webhook — update (this also re-enables and resets failures)
			if err := appStore.Webhooks.UpdateURL(ctx, userID, encrypted); err != nil {
				dest := "/settings/webhooks?error=Failed to update webhook"
				if e.Request.Header.Get("HX-Request") != "" {
					e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
					return e.HTML(http.StatusNoContent, "")
				}
				return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
			}
		}

		dest := "/settings/webhooks?success=Webhook saved successfully"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
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
		ctx := context.Background()
		if err := appStore.Webhooks.Delete(ctx, userID); err != nil {
			dest := "/settings/webhooks?error=Failed to delete webhook"
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
		}

		dest := "/settings/webhooks?success=Webhook removed"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}
