package session

import (
	"context"
	"net/http"
)

// contextKey is an unexported type for context keys in this package.
type contextKey int

const userContextKey contextKey = iota

// CookieName is the name of the session cookie.
const CookieName = "create-mod-auth"

// Middleware reads the session cookie, validates it, and populates the
// request context with the authenticated user. It replaces PocketBase's
// cookieAuth middleware.
func Middleware(store *Store, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			sess, err := store.Validate(r.Context(), cookie.Value)
			if err != nil || sess == nil {
				// Invalid or expired session -- clear the cookie
				http.SetCookie(w, &http.Cookie{
					Name:     CookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					Secure:   secure,
					SameSite: http.SameSiteLaxMode,
				})
				next.ServeHTTP(w, r)
				return
			}

			// Populate request context with the session user
			ctx := context.WithValue(r.Context(), userContextKey, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ContextWithSession returns a new context with the session stored.
// This is used by the router's cookieAuth middleware to inject the session
// into the PocketBase request context.
func ContextWithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, userContextKey, sess)
}

// FromContext retrieves the session from the request context.
// Returns nil if no session is present (unauthenticated request).
func FromContext(ctx context.Context) *Session {
	sess, _ := ctx.Value(userContextKey).(*Session)
	return sess
}

// UserFromContext is a convenience function that returns the session user.
// Returns nil if not authenticated.
func UserFromContext(ctx context.Context) *SessionUser {
	sess := FromContext(ctx)
	if sess == nil {
		return nil
	}
	return sess.User
}

// SetCookie sets the session cookie on the response.
func SetCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie clears the session cookie.
func ClearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
