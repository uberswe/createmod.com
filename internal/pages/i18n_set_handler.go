package pages

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// SetLanguageHandler sets a cookie with the selected language and redirects back.
func SetLanguageHandler() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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
		returnTo := strings.TrimSpace(e.Request.URL.Query().Get("return_to"))
		if returnTo == "" {
			returnTo = "/"
		}

		// Support HTMX redirect
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", returnTo)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusFound, returnTo)
	}
}
