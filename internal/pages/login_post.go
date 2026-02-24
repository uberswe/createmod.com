package pages

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// LoginPostHandler handles POST /login by proxying credentials to PocketBase
// and forwarding the auth cookie back to the client. Supports HTMX redirects.
func LoginPostHandler(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Parse form fields
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		identity := strings.TrimSpace(e.Request.Form.Get("username"))
		password := strings.TrimSpace(e.Request.Form.Get("password"))
		if identity == "" || password == "" {
			// re-render login page with 400
			// keep it simple: redirect back to /login with 400 for normal reqs
			if e.Request.Header.Get("HX-Request") != "" {
				// HTMX: show simple error text
				return e.String(http.StatusBadRequest, "missing credentials")
			}
			return e.Redirect(http.StatusFound, "/login")
		}

		// Build PocketBase auth endpoint URL using current host/scheme
		scheme := "http"
		if e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		host := e.Request.Host
		pbURL := scheme + "://" + host + "/api/collections/users/auth-with-password"

		payload := map[string]string{
			"identity": identity,
			"password": password,
		}
		b, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, pbURL, bytes.NewReader(b))
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to build auth request")
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return e.String(http.StatusBadGateway, "auth service unavailable")
		}
		defer resp.Body.Close()
		// Drain body to allow re-use; we don't need it
		_, _ = io.Copy(io.Discard, resp.Body)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Forward any Set-Cookie headers (auth cookie is set by PocketBase hooks)
			for _, c := range resp.Cookies() {
				http.SetCookie(e.Response, c)
			}
			returnTo := strings.TrimSpace(e.Request.Form.Get("return_to"))
			if returnTo == "" {
				returnTo = "/"
			}
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", returnTo)
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusFound, returnTo)
		}

		// Authentication failed; show login again
		if e.Request.Header.Get("HX-Request") != "" {
			return e.String(http.StatusUnauthorized, "invalid username or password")
		}
		return e.Redirect(http.StatusFound, "/login")
	}
}
