package pages

import (
	"bytes"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	tmpl "html/template"
	"image/png"
	"net/http"
	"strings"

	"createmod/internal/server"

	"github.com/pquerna/otp/totp"
)

var totpSetupTemplates = append([]string{
	"./template/totp-setup.html",
}, commonTemplates...)

var totpBackupCodesTemplates = append([]string{
	"./template/totp-backup-codes.html",
}, commonTemplates...)

type TOTPSetupData struct {
	DefaultData
	QRCodeDataURI tmpl.URL
	ManualKey     string
	Error         string
}

type TOTPBackupCodesData struct {
	DefaultData
	BackupCodes []string
}

func TOTPSetupHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()

		user, err := appStore.Users.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security"))
		}

		accountName := user.Email
		if accountName == "" {
			accountName = user.Username
		}

		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "CreateMod.com",
			AccountName: accountName,
		})
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to generate TOTP secret")
		}

		encrypted, err := encryptTOTPSecret(key.Secret())
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to encrypt secret")
		}

		err = appStore.Security.UpsertTOTP(ctx, &store.UserTOTP{
			UserID:          userID,
			SecretEncrypted: encrypted,
			Enabled:         false,
			Verified:        false,
		})
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to store TOTP")
		}

		img, err := key.Image(200, 200)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to generate QR code")
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return e.String(http.StatusInternalServerError, "failed to encode QR code")
		}
		qrDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

		d := TOTPSetupData{
			QRCodeDataURI: tmpl.URL(qrDataURI),
			ManualKey:     key.Secret(),
		}
		d.Populate(e)
		d.SettingsPage = "security"
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Set Up Two-Factor Authentication")
		d.Slug = "/settings/security/totp/setup"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(totpSetupTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func TOTPSetupVerifyHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		code := strings.TrimSpace(e.Request.Form.Get("code"))
		if code == "" {
			return renderTOTPSetupError(e, registry, cacheService, appStore, "Please enter the verification code.")
		}

		userTOTP, err := appStore.Security.GetTOTP(ctx, userID)
		if err != nil || userTOTP == nil {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security/totp/setup"))
		}

		secret, err := decryptTOTPSecret(userTOTP.SecretEncrypted)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to decrypt secret")
		}

		if !totp.Validate(code, secret) {
			return renderTOTPSetupError(e, registry, cacheService, appStore, "Invalid code. Please try again.")
		}

		if err := appStore.Security.EnableTOTP(ctx, userID); err != nil {
			return e.String(http.StatusInternalServerError, "failed to enable TOTP")
		}

		_ = appStore.Security.DeleteTOTPBackupCodes(ctx, userID)

		backupCodes := make([]string, 10)
		for i := range backupCodes {
			b := make([]byte, 4)
			_, _ = rand.Read(b)
			raw := hex.EncodeToString(b)
			backupCodes[i] = raw
			_ = appStore.Security.CreateTOTPBackupCode(ctx, userID, hashCode(raw))
		}

		settings, _ := appStore.Security.GetSecuritySettings(ctx, userID)
		if settings == nil {
			settings = &store.SecuritySettings{UserID: userID}
		}
		settings.TOTPEnabled = true
		_ = appStore.Security.UpsertSecuritySettings(ctx, settings)

		d := TOTPBackupCodesData{
			BackupCodes: backupCodes,
		}
		d.Populate(e)
		d.SettingsPage = "security"
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Backup Codes")
		d.Slug = "/settings/security"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(totpBackupCodesTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func TOTPDisableHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		code := strings.TrimSpace(e.Request.Form.Get("code"))
		if code == "" {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security"))
		}

		userTOTP, err := appStore.Security.GetTOTP(ctx, userID)
		if err != nil || userTOTP == nil {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security"))
		}

		secret, err := decryptTOTPSecret(userTOTP.SecretEncrypted)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to decrypt secret")
		}

		if !totp.Validate(code, secret) {
			if e.Request.Header.Get("HX-Request") != "" {
				return e.String(http.StatusBadRequest, "Invalid code")
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security?error=invalid_code"))
		}

		_ = appStore.Security.DisableTOTP(ctx, userID)
		_ = appStore.Security.DeleteTOTPBackupCodes(ctx, userID)

		settings, _ := appStore.Security.GetSecuritySettings(ctx, userID)
		if settings == nil {
			settings = &store.SecuritySettings{UserID: userID}
		}
		settings.TOTPEnabled = false
		_ = appStore.Security.UpsertSecuritySettings(ctx, settings)

		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security"))
	}
}

func renderTOTPSetupError(e *server.RequestEvent, registry *server.Registry, cacheService *cache.Service, appStore *store.Store, msg string) error {
	userID := authenticatedUserID(e)
	ctx := e.Request.Context()

	userTOTP, err := appStore.Security.GetTOTP(ctx, userID)
	if err != nil || userTOTP == nil {
		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security/totp/setup"))
	}

	secret, err := decryptTOTPSecret(userTOTP.SecretEncrypted)
	if err != nil {
		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security/totp/setup"))
	}

	user, _ := appStore.Users.GetUserByID(ctx, userID)
	accountName := ""
	if user != nil {
		accountName = user.Email
		if accountName == "" {
			accountName = user.Username
		}
	}
	if accountName == "" {
		accountName = userID
	}

	rawSecret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security/totp/setup"))
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "CreateMod.com",
		AccountName: accountName,
		Secret:      rawSecret,
	})
	if err != nil {
		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security/totp/setup"))
	}

	img, _ := key.Image(200, 200)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	qrDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	d := TOTPSetupData{
		QRCodeDataURI: tmpl.URL(qrDataURI),
		ManualKey:     secret,
		Error:         msg,
	}
	d.Populate(e)
	d.SettingsPage = "security"
	d.HideOutstream = true
	d.Title = i18n.T(d.Language, "Set Up Two-Factor Authentication")
	d.Slug = "/settings/security/totp/setup"
	d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

	html, err := registry.LoadFiles(totpSetupTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}
