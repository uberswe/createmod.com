package pages

import (
	"createmod/internal/auth"
	"createmod/internal/session"
	"createmod/internal/store"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// LoginPostHandler handles POST /login by authenticating against PostgreSQL
// and creating a session.
func LoginPostHandler(app *pocketbase.PocketBase, appStore *store.Store, sessStore *session.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Parse form fields
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		identity := strings.TrimSpace(e.Request.Form.Get("username"))
		password := strings.TrimSpace(e.Request.Form.Get("password"))
		if identity == "" || password == "" {
			if e.Request.Header.Get("HX-Request") != "" {
				return e.String(http.StatusBadRequest, "missing credentials")
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
		}

		return loginWithStore(e, appStore, sessStore, identity, password)
	}
}

// loginWithStore authenticates against the PostgreSQL database directly.
func loginWithStore(e *core.RequestEvent, appStore *store.Store, sessStore *session.Store, identity, password string) error {
	ctx := e.Request.Context()

	// Try to find user by email first, then by username
	user, err := appStore.Users.GetUserByEmail(ctx, identity)
	if err != nil || user == nil {
		user, err = appStore.Users.GetUserByUsername(ctx, identity)
	}
	if err != nil || user == nil {
		return loginFailed(e)
	}

	// Check if user is deleted
	if user.Deleted != nil {
		return loginFailed(e)
	}

	// Verify password (bcrypt primary, phpass legacy fallback)
	matched, needsRehash := auth.CheckPassword(user.PasswordHash, user.OldPassword, password)
	if !matched {
		return loginFailed(e)
	}

	// Auto-rehash legacy phpass password to bcrypt
	if needsRehash {
		if newHash, err := auth.HashPassword(password); err == nil {
			_ = appStore.Users.UpdateUserPassword(ctx, user.ID, newHash)
		}
	}

	// Create session
	token, err := sessStore.Create(ctx, user.ID)
	if err != nil {
		return e.String(http.StatusInternalServerError, "failed to create session")
	}

	// Set session cookie
	secure := e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")
	session.SetCookie(e.Response, token, secure)

	return loginSuccess(e)
}


func loginSuccess(e *core.RequestEvent) error {
	returnTo := strings.TrimSpace(e.Request.Form.Get("return_to"))
	if returnTo == "" {
		returnTo = "/"
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo))
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusFound, LangRedirectURL(e, returnTo))
}

func loginFailed(e *core.RequestEvent) error {
	if e.Request.Header.Get("HX-Request") != "" {
		return e.String(http.StatusUnauthorized, "invalid username or password")
	}
	return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
}
