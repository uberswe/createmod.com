package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/session"
	"createmod/internal/store"
	"net/http"
	"strings"

	"createmod/internal/server"

	"github.com/pquerna/otp/totp"
)

var totpChallengeTemplates = append([]string{
	"./template/totp-challenge.html",
}, commonTemplates...)

type TOTPChallengeData struct {
	DefaultData
	Error string
}

func TOTPChallengeHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		pending, err := readPendingAuth(e)
		if err != nil || pending == nil {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
		}

		d := TOTPChallengeData{}
		d.Populate(e)
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Two-Factor Authentication")
		d.Slug = "/auth/totp"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(totpChallengeTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func TOTPChallengeVerifyHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, sessStore *session.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		pending, err := readPendingAuth(e)
		if err != nil || pending == nil {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
		}

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		code := strings.TrimSpace(e.Request.Form.Get("code"))
		if code == "" {
			return renderTOTPChallengeError(e, registry, cacheService, appStore, i18n.T(preferredLanguageFromRequest(e.Request), "Please enter your authentication code."))
		}

		ctx := e.Request.Context()

		userTOTP, err := appStore.Security.GetTOTP(ctx, pending.UserID)
		if err != nil || userTOTP == nil {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
		}

		secret, err := decryptTOTPSecret(userTOTP.SecretEncrypted)
		if err != nil {
			return e.String(http.StatusInternalServerError, "security error")
		}

		if totp.Validate(code, secret) {
			return completeChallenge(e, appStore, sessStore, pending, "totp")
		}

		codeHash := hashCode(code)
		backupCodes, _ := appStore.Security.ListTOTPBackupCodes(ctx, pending.UserID)
		for _, bc := range backupCodes {
			if !bc.Used && bc.CodeHash == codeHash {
				_ = appStore.Security.MarkBackupCodeUsed(ctx, bc.ID)
				return completeChallenge(e, appStore, sessStore, pending, "totp")
			}
		}

		return renderTOTPChallengeError(e, registry, cacheService, appStore, i18n.T(preferredLanguageFromRequest(e.Request), "Invalid code. Please try again."))
	}
}

func renderTOTPChallengeError(e *server.RequestEvent, registry *server.Registry, cacheService *cache.Service, appStore *store.Store, msg string) error {
	d := TOTPChallengeData{
		Error: msg,
	}
	d.Populate(e)
	d.HideOutstream = true
	d.Title = i18n.T(d.Language, "Two-Factor Authentication")
	d.Slug = "/auth/totp"
	d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

	html, err := registry.LoadFiles(totpChallengeTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}
