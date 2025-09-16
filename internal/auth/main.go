package auth

import "net/http"

const CookieName = "create-mod-auth"

// ClearAuthCookie clears the auth cookie by setting MaxAge=0 and an expired date.
// secure: set to true in production (HTTPS), false in local dev.
func ClearAuthCookie(w http.ResponseWriter, secure bool) {
	// Expire the cookie
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
