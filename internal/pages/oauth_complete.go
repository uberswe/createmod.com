package pages

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/session"
	"createmod/internal/store"

	"github.com/drexedam/gravatar"
)

// oauthCompleteTemplate is the email-collection step shown after an OAuth
// provider returns no email (e.g. a GitHub user with a private primary email).
const oauthCompleteTemplate = "./template/oauth_complete.html"

var oauthCompleteTemplates = append([]string{
	oauthCompleteTemplate,
}, commonTemplates...)

// oauthPendingCookie is the cookie that carries the HMAC-signed pending
// OAuth claim between the callback and the complete-registration form.
const (
	oauthPendingCookie = "oauth-pending"
	oauthPendingTTL    = 10 * time.Minute
)

// oauthSigningSecret is set at startup via SetOAuthSigningSecret. It signs
// the pending-OAuth cookie so the payload can safely round-trip through
// the user's browser (and survive pod switches across replicas).
var oauthSigningSecret []byte

// SetOAuthSigningSecret stores the secret used to HMAC-sign the pending
// OAuth state cookie. Called once from router.Register at startup.
func SetOAuthSigningSecret(secret string) {
	oauthSigningSecret = []byte(secret)
}

// oauthPending is the payload encoded into the oauth-pending cookie after
// an OAuth callback succeeds but the account cannot be created (no email).
type oauthPending struct {
	Provider   string `json:"p"`
	ProviderID string `json:"pid"`
	Username   string `json:"u"`
	Avatar     string `json:"a"`
	Exp        int64  `json:"e"` // unix seconds
}

// encodeOAuthPending returns a "<payload-b64>.<sig-b64>" token.
func encodeOAuthPending(p oauthPending) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, oauthSigningSecret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}

// decodeOAuthPending verifies the HMAC and expiry before returning the payload.
func decodeOAuthPending(token string) (*oauthPending, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("malformed token")
	}
	mac := hmac.New(sha256.New, oauthSigningSecret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("signature mismatch")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var p oauthPending
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Exp > 0 && time.Now().Unix() > p.Exp {
		return nil, errors.New("token expired")
	}
	return &p, nil
}

// setOAuthPendingCookie writes a signed cookie with the pending-OAuth payload.
func setOAuthPendingCookie(e *server.RequestEvent, p oauthPending) error {
	p.Exp = time.Now().Add(oauthPendingTTL).Unix()
	token, err := encodeOAuthPending(p)
	if err != nil {
		return err
	}
	http.SetCookie(e.Response, &http.Cookie{
		Name:     oauthPendingCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(oauthPendingTTL.Seconds()),
		HttpOnly: true,
		Secure:   e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https"),
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// clearOAuthPendingCookie removes the pending-OAuth cookie.
func clearOAuthPendingCookie(e *server.RequestEvent) {
	http.SetCookie(e.Response, &http.Cookie{
		Name:     oauthPendingCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// OAuthCompleteData drives the oauth_complete.html template.
type OAuthCompleteData struct {
	DefaultData
	Provider        string // pretty provider name (e.g. "GitHub")
	SuggestedUser   string
	EmailPrefill    string
	Error           string
}

func prettyProviderName(p string) string {
	switch p {
	case "github":
		return "GitHub"
	case "discord":
		return "Discord"
	}
	if p == "" {
		return ""
	}
	return strings.ToUpper(p[:1]) + p[1:]
}

func readPendingFromCookie(e *server.RequestEvent) (*oauthPending, error) {
	cookie, err := e.Request.Cookie(oauthPendingCookie)
	if err != nil || cookie.Value == "" {
		return nil, errors.New("no pending cookie")
	}
	return decodeOAuthPending(cookie.Value)
}

// OAuthCompleteHandler renders the "please provide an email" form shown when
// an OAuth provider returned no usable email address.
func OAuthCompleteHandler(registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		pending, err := readPendingFromCookie(e)
		if err != nil {
			return oauthLoginErrorRedirect(e, "state_missing")
		}

		d := OAuthCompleteData{
			Provider:      prettyProviderName(pending.Provider),
			SuggestedUser: pending.Username,
		}
		d.Populate(e)
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Complete your account")
		d.Slug = "/auth/oauth/complete"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"

		html, err := registry.LoadFiles(oauthCompleteTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// OAuthCompletePostHandler creates the account + OAuth link using the email
// submitted by the user, then logs them in.
func OAuthCompletePostHandler(registry *server.Registry, appStore *store.Store, sessStore *session.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		pending, err := readPendingFromCookie(e)
		if err != nil {
			return oauthLoginErrorRedirect(e, "state_missing")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		email := strings.TrimSpace(strings.ToLower(e.Request.FormValue("email")))
		if !isLikelyEmail(email) {
			return renderOAuthCompleteWithError(e, registry, pending, email, i18n.T(preferredLanguageFromRequest(e.Request), "Please enter a valid email address."))
		}

		ctx := e.Request.Context()

		// If an account with this email already exists, link the OAuth
		// identity to that account instead of creating a new row. This is
		// a convenience for users who registered with email previously.
		var userID string
		existing, _ := appStore.Users.GetUserByEmail(ctx, email)
		if existing != nil && existing.Deleted == nil {
			userID = existing.ID
		}

		if userID == "" {
			username := sanitizeUsername(pending.Username)
			if username == "" {
				username = pending.Provider + "_" + pending.ProviderID
			}
			username = ensureUniqueUsername(ctx, appStore, username)

			avatarURL := pending.Avatar
			if avatarURL == "" {
				avatarURL = gravatar.New(email).
					Size(200).
					Default(gravatar.MysteryMan).
					Rating(gravatar.Pg).
					AvatarURL()
			}

			newUser := &store.User{
				Email:    email,
				Username: username,
				Avatar:   avatarURL,
				Verified: true,
			}
			if err := appStore.Users.CreateUser(ctx, newUser); err != nil {
				slog.Error("oauth complete: user create failed", "error", err)
				return renderOAuthCompleteWithError(e, registry, pending, email, i18n.T(preferredLanguageFromRequest(e.Request), "That email is already in use. Please sign in with it instead."))
			}
			userID = newUser.ID
		}

		// Link the OAuth identity to the resolved account.
		if err := appStore.Auth.Create(ctx, &store.ExternalAuth{
			UserID:     userID,
			Provider:   pending.Provider,
			ProviderID: pending.ProviderID,
		}); err != nil {
			slog.Error("oauth complete: auth link failed", "error", err)
			// Continue — user will be signed in, they can try the link again.
		}

		clearOAuthPendingCookie(e)
		return oauthCreateSession(e, appStore, sessStore, userID)
	}
}

func renderOAuthCompleteWithError(e *server.RequestEvent, registry *server.Registry, pending *oauthPending, email, msg string) error {
	d := OAuthCompleteData{
		Provider:      prettyProviderName(pending.Provider),
		SuggestedUser: pending.Username,
		EmailPrefill:  email,
		Error:         msg,
	}
	d.Populate(e)
	d.HideOutstream = true
	d.Title = i18n.T(d.Language, "Complete your account")
	html, err := registry.LoadFiles(oauthCompleteTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}

// isLikelyEmail returns true if the input has a minimal email shape — we
// rely on Go's net/mail for parsing rather than rolling our own regex.
func isLikelyEmail(s string) bool {
	if s == "" || len(s) > 320 {
		return false
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return false
	}
	return true
}
