package pages

import (
	"net/http"
	"strings"

	"createmod/internal/server"
)

// SetLanguageHandler sets a cookie with the selected language and redirects
// to the language-prefixed URL for the requested page.
func SetLanguageHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		// get lang param and validate
		l := strings.TrimSpace(e.Request.URL.Query().Get("l"))
		if !isSupportedLanguage(l) {
			l = "en"
		}
		// cookie details
		secure := e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")
		cookie := &http.Cookie{
			Name:     "cm_lang",
			Value:    l,
			Path:     "/",
			MaxAge:   31536000, // 1 year
			HttpOnly: false,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(e.Response, cookie)

		// compute return_to target; default "/"
		returnTo := safeRedirectPath(e.Request.URL.Query().Get("return_to"), "/")

		// Strip any existing language prefix from returnTo, then re-prefix
		// with the newly selected language.
		_, barePath := StripLangPrefix(returnTo)
		if barePath == "" {
			barePath = "/"
		}
		target := PrefixedPath(l, barePath)

		// Support HTMX redirect
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", target)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusFound, target)
	}
}
