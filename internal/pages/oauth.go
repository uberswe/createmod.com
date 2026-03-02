package pages

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"createmod/internal/auth"
	"createmod/internal/session"
	"createmod/internal/store"

	"github.com/drexedam/gravatar"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// OAuthRedirectHandler initiates the OAuth flow by redirecting to the provider.
func OAuthRedirectHandler(provider *auth.OAuthProvider) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if provider == nil {
			return e.String(http.StatusNotFound, "OAuth provider not configured")
		}

		// Generate random state
		stateBytes := make([]byte, 16)
		if _, err := rand.Read(stateBytes); err != nil {
			return e.String(http.StatusInternalServerError, "failed to generate state")
		}
		state := hex.EncodeToString(stateBytes)

		// Store state in cookie (10 min TTL)
		http.SetCookie(e.Response, &http.Cookie{
			Name:     "oauth-state",
			Value:    state,
			Path:     "/",
			MaxAge:   600,
			HttpOnly: true,
			Secure:   e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https"),
			SameSite: http.SameSiteLaxMode,
		})

		return e.Redirect(http.StatusFound, provider.AuthURL(state))
	}
}

// OAuthCallbackHandler handles the OAuth callback, creating or linking user accounts.
func OAuthCallbackHandler(app *pocketbase.PocketBase, provider *auth.OAuthProvider, appStore *store.Store, sessStore *session.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if provider == nil {
			return e.String(http.StatusNotFound, "OAuth provider not configured")
		}

		// Validate state
		stateCookie, err := e.Request.Cookie("oauth-state")
		if err != nil || stateCookie.Value == "" {
			return e.Redirect(http.StatusFound, "/login")
		}

		queryState := e.Request.URL.Query().Get("state")
		if queryState == "" || queryState != stateCookie.Value {
			return e.Redirect(http.StatusFound, "/login")
		}

		// Clear state cookie
		http.SetCookie(e.Response, &http.Cookie{
			Name:     "oauth-state",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		// Check for error from provider
		if errCode := e.Request.URL.Query().Get("error"); errCode != "" {
			app.Logger().Warn("OAuth error from provider", "provider", provider.Name, "error", errCode)
			return e.Redirect(http.StatusFound, "/login")
		}

		// Exchange code for token
		code := e.Request.URL.Query().Get("code")
		if code == "" {
			return e.Redirect(http.StatusFound, "/login")
		}

		token, err := provider.Exchange(e.Request.Context(), code)
		if err != nil {
			app.Logger().Error("OAuth token exchange failed", "provider", provider.Name, "error", err)
			return e.Redirect(http.StatusFound, "/login")
		}

		// Fetch user info from provider
		oauthUser, err := provider.FetchUser(e.Request.Context(), token)
		if err != nil {
			app.Logger().Error("OAuth user fetch failed", "provider", provider.Name, "error", err)
			return e.Redirect(http.StatusFound, "/login")
		}

		ctx := e.Request.Context()

		// Look up existing external auth link
		extAuth, err := appStore.Auth.GetByProvider(ctx, provider.Name, oauthUser.ID)
		if err == nil && extAuth != nil {
			// Existing link found -- log the user in
			return oauthCreateSession(e, appStore, sessStore, extAuth.UserID)
		}

		// No existing link -- check if user with same email exists
		var userID string
		if oauthUser.Email != "" {
			existingUser, _ := appStore.Users.GetUserByEmail(ctx, oauthUser.Email)
			if existingUser != nil && existingUser.Deleted == nil {
				userID = existingUser.ID
			}
		}

		if userID == "" {
			// Create new user
			username := sanitizeUsername(oauthUser.Username)
			if username == "" {
				username = fmt.Sprintf("%s_%s", provider.Name, oauthUser.ID)
			}

			// Ensure username uniqueness
			username = ensureUniqueUsername(ctx, appStore, username)

			avatarURL := oauthUser.Avatar
			if avatarURL == "" && oauthUser.Email != "" {
				avatarURL = gravatar.New(oauthUser.Email).
					Size(200).
					Default(gravatar.MysteryMan).
					Rating(gravatar.Pg).
					AvatarURL()
			}

			newUser := &store.User{
				Email:    oauthUser.Email,
				Username: username,
				Avatar:   avatarURL,
				Verified: oauthUser.Email != "",
			}
			if err := appStore.Users.CreateUser(ctx, newUser); err != nil {
				app.Logger().Error("OAuth user creation failed", "error", err)
				return e.Redirect(http.StatusFound, "/login")
			}
			userID = newUser.ID

			// Sync to PocketBase (transition bridge)
			syncUserToPocketBase(app, newUser)
		}

		// Create external auth link
		if err := appStore.Auth.Create(ctx, &store.ExternalAuth{
			UserID:     userID,
			Provider:   provider.Name,
			ProviderID: oauthUser.ID,
		}); err != nil {
			app.Logger().Error("OAuth auth link creation failed", "error", err)
			// Still log the user in even if linking fails
		}

		return oauthCreateSession(e, appStore, sessStore, userID)
	}
}

// oauthCreateSession creates a session and redirects to home.
func oauthCreateSession(e *core.RequestEvent, appStore *store.Store, sessStore *session.Store, userID string) error {
	ctx := e.Request.Context()

	// Verify user still exists and isn't deleted
	user, err := appStore.Users.GetUserByID(ctx, userID)
	if err != nil || user == nil || user.Deleted != nil {
		return e.Redirect(http.StatusFound, "/login")
	}

	token, err := sessStore.Create(ctx, userID)
	if err != nil {
		return e.String(http.StatusInternalServerError, "failed to create session")
	}

	secure := e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")
	session.SetCookie(e.Response, token, secure)

	return e.Redirect(http.StatusFound, LangRedirectURL(e, "/"))
}

// sanitizeUsername removes non-alphanumeric characters and lowercases.
func sanitizeUsername(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ensureUniqueUsername appends random chars if the username is already taken.
func ensureUniqueUsername(ctx context.Context, appStore *store.Store, username string) string {
	candidate := username
	for i := 0; i < 10; i++ {
		existing, _ := appStore.Users.GetUserByUsername(ctx, candidate)
		if existing == nil {
			return candidate
		}
		suffix := make([]byte, 3)
		rand.Read(suffix)
		candidate = username + hex.EncodeToString(suffix)[:4]
	}
	return candidate
}

// syncUserToPocketBase creates a matching user record in PocketBase
// during the transition period so PB hooks (comments, ratings, etc.) still work.
func syncUserToPocketBase(app *pocketbase.PocketBase, user *store.User) {
	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		app.Logger().Warn("failed to find PB users collection for sync", "error", err)
		return
	}

	record := core.NewRecord(collection)
	if user.ID != "" {
		record.Set("id", user.ID)
	}
	record.Set("username", user.Username)
	record.Set("email", user.Email)
	record.Set("emailVisibility", false)
	record.SetVerified(user.Verified)

	avatarURL := user.Avatar
	if avatarURL == "" && user.Email != "" {
		avatarURL = gravatar.New(user.Email).
			Size(200).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
	}
	record.Set("avatar", avatarURL)

	// Set a random password so PB doesn't reject the record
	randPass := make([]byte, 16)
	rand.Read(randPass)
	record.SetPassword(hex.EncodeToString(randPass))

	record.Set("created", time.Now())
	record.Set("updated", time.Now())

	if err := app.Save(record); err != nil {
		app.Logger().Warn("failed to sync user to PocketBase", "error", err, "user_id", user.ID)
	}
}
