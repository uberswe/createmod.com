package pages

import (
	"createmod/internal/session"
	"createmod/internal/store"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"createmod/internal/server"
)

const (
	pendingAuthCookieName = "auth-pending"
	pendingAuthTTL        = 10 * time.Minute
)

type pendingAuth struct {
	UserID   string `json:"u"`
	IP       string `json:"ip"`
	ReturnTo string `json:"r"`
	Needs    string `json:"n"`
	Exp      int64  `json:"e"`
}

func encodePendingAuth(p pendingAuth) (string, error) {
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

func decodePendingAuth(token string) (*pendingAuth, error) {
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
	var p pendingAuth
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Exp > 0 && time.Now().Unix() > p.Exp {
		return nil, errors.New("token expired")
	}
	return &p, nil
}

func setPendingAuthCookie(e *server.RequestEvent, p pendingAuth) error {
	p.Exp = time.Now().Add(pendingAuthTTL).Unix()
	token, err := encodePendingAuth(p)
	if err != nil {
		return err
	}
	http.SetCookie(e.Response, &http.Cookie{
		Name:     pendingAuthCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(pendingAuthTTL.Seconds()),
		HttpOnly: true,
		Secure:   e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https"),
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func clearPendingAuthCookie(e *server.RequestEvent) {
	http.SetCookie(e.Response, &http.Cookie{
		Name:     pendingAuthCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func readPendingAuth(e *server.RequestEvent) (*pendingAuth, error) {
	cookie, err := e.Request.Cookie(pendingAuthCookieName)
	if err != nil || cookie.Value == "" {
		return nil, errors.New("no pending auth cookie")
	}
	return decodePendingAuth(cookie.Value)
}

func removeNeed(needs, completed string) string {
	parts := strings.Split(needs, ",")
	var remaining []string
	for _, p := range parts {
		if p != "" && p != completed {
			remaining = append(remaining, p)
		}
	}
	return strings.Join(remaining, ",")
}

func nextChallenge(needs string) string {
	parts := strings.Split(needs, ",")
	for _, p := range parts {
		if p != "" {
			return p
		}
	}
	return ""
}

func challengeURL(challenge string) string {
	switch challenge {
	case "totp":
		return "/auth/totp"
	case "ip":
		return "/auth/verify-ip"
	default:
		return "/login"
	}
}

func completeChallenge(e *server.RequestEvent, appStore *store.Store, sessStore *session.Store, pending *pendingAuth, completed string) error {
	remaining := removeNeed(pending.Needs, completed)

	if next := nextChallenge(remaining); next != "" {
		pending.Needs = remaining
		if err := setPendingAuthCookie(e, *pending); err != nil {
			return e.String(http.StatusInternalServerError, "security check failed")
		}
		url := challengeURL(next)
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, url))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusFound, LangRedirectURL(e, url))
	}

	clearPendingAuthCookie(e)

	ctx := e.Request.Context()
	token, err := sessStore.Create(ctx, pending.UserID)
	if err != nil {
		return e.String(http.StatusInternalServerError, "failed to create session")
	}

	secure := e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")
	session.SetCookie(e.Response, token, secure)

	returnTo := pending.ReturnTo
	if returnTo == "" {
		returnTo = "/"
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo))
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusFound, LangRedirectURL(e, returnTo))
}
