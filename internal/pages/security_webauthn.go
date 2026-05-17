package pages

import (
	"createmod/internal/store"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"createmod/internal/server"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var webAuthnInstance *webauthn.WebAuthn

const (
	webauthnSessionCookie = "webauthn-session"
	webauthnSessionTTL    = 5 * time.Minute
)

func InitWebAuthn() {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8090"
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		slog.Warn("webauthn: invalid BASE_URL, disabling passkeys", "error", err)
		return
	}

	rpID := parsed.Hostname()
	rpOrigin := parsed.Scheme + "://" + parsed.Host

	wconfig := &webauthn.Config{
		RPDisplayName: "CreateMod.com",
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
	}
	webAuthnInstance, err = webauthn.New(wconfig)
	if err != nil {
		slog.Error("webauthn: failed to initialize", "error", err)
	}
}

func WebAuthnEnabled() bool {
	return webAuthnInstance != nil
}

type webAuthnUser struct {
	id       string
	name     string
	passkeys []store.Passkey
}

func (u *webAuthnUser) WebAuthnID() []byte          { return []byte(u.id) }
func (u *webAuthnUser) WebAuthnName() string         { return u.name }
func (u *webAuthnUser) WebAuthnDisplayName() string   { return u.name }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(u.passkeys))
	for i, pk := range u.passkeys {
		transports := make([]protocol.AuthenticatorTransport, len(pk.Transport))
		for j, t := range pk.Transport {
			transports[j] = protocol.AuthenticatorTransport(t)
		}
		creds[i] = webauthn.Credential{
			ID:              pk.CredentialID,
			PublicKey:       pk.PublicKey,
			AttestationType: pk.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    pk.AAGUID,
				SignCount: uint32(pk.SignCount),
			},
		}
	}
	return creds
}

type webAuthnSessionWrapper struct {
	Session *webauthn.SessionData `json:"s"`
	Exp     int64                 `json:"e"`
}

func encodeWebAuthnSession(session *webauthn.SessionData) (string, error) {
	wrapper := webAuthnSessionWrapper{
		Session: session,
		Exp:     time.Now().Add(webauthnSessionTTL).Unix(),
	}
	raw, err := json.Marshal(wrapper)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, webauthnSigningSecret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}

func decodeWebAuthnSession(token string) (*webauthn.SessionData, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("malformed token")
	}
	mac := hmac.New(sha256.New, webauthnSigningSecret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("signature mismatch")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var wrapper webAuthnSessionWrapper
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, err
	}
	if time.Now().Unix() > wrapper.Exp {
		return nil, errors.New("session expired")
	}
	return wrapper.Session, nil
}

func setWebAuthnSessionCookie(e *server.RequestEvent, session *webauthn.SessionData) error {
	token, err := encodeWebAuthnSession(session)
	if err != nil {
		return err
	}
	http.SetCookie(e.Response, &http.Cookie{
		Name:     webauthnSessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(webauthnSessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https"),
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func readWebAuthnSession(e *server.RequestEvent) (*webauthn.SessionData, error) {
	cookie, err := e.Request.Cookie(webauthnSessionCookie)
	if err != nil || cookie.Value == "" {
		return nil, errors.New("no webauthn session")
	}
	return decodeWebAuthnSession(cookie.Value)
}

func clearWebAuthnSessionCookie(e *server.RequestEvent) {
	http.SetCookie(e.Response, &http.Cookie{
		Name:     webauthnSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
