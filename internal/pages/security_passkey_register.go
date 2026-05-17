package pages

import (
	"createmod/internal/session"
	"createmod/internal/store"
	"log/slog"
	"net/http"
	"strings"

	"createmod/internal/server"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

func PasskeyBeginRegistrationHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !WebAuthnEnabled() {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "passkeys not configured"})
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()

		sessUser := session.UserFromContext(ctx)
		if sessUser == nil {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		}

		existingPasskeys, _ := appStore.Security.ListPasskeys(ctx, userID)
		user := &webAuthnUser{id: userID, name: sessUser.Username, passkeys: existingPasskeys}

		creation, waSession, err := webAuthnInstance.BeginRegistration(user)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to begin registration"})
		}

		if err := setWebAuthnSessionCookie(e, waSession); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store session"})
		}

		return e.JSON(http.StatusOK, creation)
	}
}

func PasskeyFinishRegistrationHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !WebAuthnEnabled() {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "passkeys not configured"})
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()

		waSession, err := readWebAuthnSession(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "no registration session"})
		}
		clearWebAuthnSessionCookie(e)

		sessUser := session.UserFromContext(ctx)
		if sessUser == nil {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		}

		existingPasskeys, _ := appStore.Security.ListPasskeys(ctx, userID)
		user := &webAuthnUser{id: userID, name: sessUser.Username, passkeys: existingPasskeys}

		credential, err := webAuthnInstance.FinishRegistration(user, *waSession, e.Request)
		if err != nil {
			slog.Error("webauthn: finish registration failed", "error", err, "user", userID)
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "registration verification failed: " + err.Error()})
		}

		friendlyName := e.Request.URL.Query().Get("name")
		if friendlyName == "" {
			friendlyName = "Passkey"
		}

		transports := make([]string, len(credential.Transport))
		for i, t := range credential.Transport {
			transports[i] = string(t)
		}

		pk := &store.Passkey{
			UserID:          userID,
			CredentialID:    credential.ID,
			PublicKey:       credential.PublicKey,
			AttestationType: credential.AttestationType,
			Transport:       transports,
			AAGUID:          credential.Authenticator.AAGUID,
			SignCount:       int(credential.Authenticator.SignCount),
			FriendlyName:    friendlyName,
		}
		if err := appStore.Security.CreatePasskey(ctx, pk); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store passkey"})
		}

		settings, _ := appStore.Security.GetSecuritySettings(ctx, userID)
		if settings == nil {
			settings = &store.SecuritySettings{UserID: userID}
		}
		settings.PasskeysEnabled = true
		_ = appStore.Security.UpsertSecuritySettings(ctx, settings)

		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

func PasskeyDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()
		passkeyID := e.Request.PathValue("id")

		_ = appStore.Security.DeletePasskey(ctx, passkeyID, userID)

		remaining, _ := appStore.Security.ListPasskeys(ctx, userID)
		if len(remaining) == 0 {
			settings, _ := appStore.Security.GetSecuritySettings(ctx, userID)
			if settings == nil {
				settings = &store.SecuritySettings{UserID: userID}
			}
			settings.PasskeysEnabled = false
			_ = appStore.Security.UpsertSecuritySettings(ctx, settings)
		}

		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security"))
	}
}

func PasskeyDiscoverableLoginBeginHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !WebAuthnEnabled() {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "passkeys not configured"})
		}

		options, waSession, err := webAuthnInstance.BeginDiscoverableLogin()
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to begin login"})
		}

		if err := setWebAuthnSessionCookie(e, waSession); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store session"})
		}

		return e.JSON(http.StatusOK, options)
	}
}

func PasskeyLoginFinishHandler(appStore *store.Store, sessStore *session.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !WebAuthnEnabled() {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "passkeys not configured"})
		}

		waSession, err := readWebAuthnSession(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "no login session"})
		}
		clearWebAuthnSessionCookie(e)

		ctx := e.Request.Context()

		handler := func(rawID, userHandle []byte) (webauthn.User, error) {
			pk, err := appStore.Security.GetPasskeyByCredentialID(ctx, rawID)
			if err != nil || pk == nil {
				return nil, err
			}
			user, err := appStore.Users.GetUserByID(ctx, pk.UserID)
			if err != nil || user == nil || user.Deleted != nil {
				return nil, err
			}
			allPasskeys, _ := appStore.Security.ListPasskeys(ctx, pk.UserID)
			return &webAuthnUser{id: pk.UserID, name: user.Username, passkeys: allPasskeys}, nil
		}

		parsedResponse, err := protocol.ParseCredentialRequestResponse(e.Request)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid assertion response"})
		}

		credential, err := webAuthnInstance.ValidateDiscoverableLogin(handler, *waSession, parsedResponse)
		if err != nil {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "login verification failed"})
		}

		pk, err := appStore.Security.GetPasskeyByCredentialID(ctx, credential.ID)
		if err != nil || pk == nil {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unknown credential"})
		}

		_ = appStore.Security.UpdatePasskeySignCount(ctx, pk.ID, int(credential.Authenticator.SignCount))

		user, err := appStore.Users.GetUserByID(ctx, pk.UserID)
		if err != nil || user == nil || user.Deleted != nil {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "user not found"})
		}

		token, err := sessStore.Create(ctx, pk.UserID)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		}

		secure := e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")
		session.SetCookie(e.Response, token, secure)

		return e.JSON(http.StatusOK, map[string]string{"status": "ok", "redirect": "/"})
	}
}
