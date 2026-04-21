package pages

import (
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const loginTemplate = "./template/login.html"

var loginTemplates = append([]string{
	loginTemplate,
}, commonTemplates...)

type LoginData struct {
	DefaultData
}

func LoginHandler(registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := LoginData{}
		d.Populate(e)
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Login")
		d.Description = i18n.T(d.Language, "page.login.description")
		d.Slug = "/login"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.OAuthError = oauthErrorMessage(d.Language, e.Request.URL.Query().Get("oauth_error"))
		html, err := registry.LoadFiles(loginTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// oauthErrorMessage maps an oauth_error query param to a human-readable,
// translated message. Unknown or empty codes return an empty string so the
// template can suppress the alert.
func oauthErrorMessage(lang, code string) string {
	switch code {
	case "":
		return ""
	case "not_configured":
		return i18n.T(lang, "Login with this provider is not available right now.")
	case "no_email":
		return i18n.T(lang, "We could not read a verified email from the provider. Make your email public on the provider and try again.")
	case "state_missing", "state_mismatch":
		return i18n.T(lang, "Your sign-in session expired. Please try again.")
	case "provider_error", "token_exchange", "user_fetch", "missing_code":
		return i18n.T(lang, "The sign-in provider rejected the request. Please try again.")
	case "user_create", "user_missing":
		return i18n.T(lang, "We could not create or load your account from the provider. Try logging in with email instead.")
	}
	return i18n.T(lang, "Sign-in failed. Please try again.")
}
