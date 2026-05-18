package pages

import (
	"createmod/internal/auth"
	"createmod/internal/mailer"
	"createmod/internal/session"
	"createmod/internal/store"
	"net/http"
	"strings"

	"createmod/internal/server"
)

// LoginPostHandler handles POST /login by authenticating against PostgreSQL
// and creating a session.
func LoginPostHandler(appStore *store.Store, sessStore *session.Store, mailService *mailer.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		identity := strings.TrimSpace(e.Request.Form.Get("username"))
		password := strings.TrimSpace(e.Request.Form.Get("password"))
		if identity == "" || password == "" {
			if e.Request.Header.Get("HX-Request") != "" {
				return e.String(http.StatusBadRequest, "missing credentials")
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login?error=credentials"))
		}

		return loginWithStore(e, appStore, sessStore, mailService, identity, password)
	}
}

func loginWithStore(e *server.RequestEvent, appStore *store.Store, sessStore *session.Store, mailService *mailer.Service, identity, password string) error {
	ctx := e.Request.Context()

	user, err := appStore.Users.GetUserByEmail(ctx, identity)
	if err != nil || user == nil {
		user, err = appStore.Users.GetUserByUsername(ctx, identity)
	}
	if err != nil || user == nil {
		return loginFailed(e)
	}

	if user.Deleted != nil {
		return loginFailed(e)
	}

	matched, needsRehash := auth.CheckPassword(user.PasswordHash, user.OldPassword, password)
	if !matched {
		return loginFailed(e)
	}

	if needsRehash {
		if newHash, err := auth.HashPassword(password); err == nil {
			_ = appStore.Users.UpdateUserPassword(ctx, user.ID, newHash)
		}
	}

	returnTo := safeRedirectPath(e.Request.Form.Get("return_to"), "/")
	return maybeCreateSessionOrChallenge(e, appStore, sessStore, mailService, user.ID, returnTo)
}


func loginFailed(e *server.RequestEvent) error {
	if e.Request.Header.Get("HX-Request") != "" {
		return e.String(http.StatusUnauthorized, "invalid username or password")
	}
	return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login?error=credentials"))
}
