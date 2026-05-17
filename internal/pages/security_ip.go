package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/mailer"
	"createmod/internal/session"
	"createmod/internal/store"
	"crypto/subtle"
	"net/http"
	"strings"

	"createmod/internal/server"
)

var ipVerifyTemplates = append([]string{
	"./template/ip-verify.html",
}, commonTemplates...)

type IPVerifyData struct {
	DefaultData
	Error   string
	Success string
}

func IPVerificationToggleHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}

		enabled := e.Request.Form.Get("enabled") == "on" || e.Request.Form.Get("enabled") == "true"

		settings, _ := appStore.Security.GetSecuritySettings(ctx, userID)
		if settings == nil {
			settings = &store.SecuritySettings{UserID: userID}
		}
		settings.NewIPVerification = enabled
		_ = appStore.Security.UpsertSecuritySettings(ctx, settings)

		if enabled {
			_ = appStore.Security.UpsertKnownIP(ctx, &store.KnownIP{
				UserID:    userID,
				IPAddress: e.RealIP(),
				UserAgent: e.Request.UserAgent(),
				Verified:  true,
			})
		}

		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security"))
	}
}

func IPVerificationChallengeHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, mailService *mailer.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		pending, err := readPendingAuth(e)
		if err != nil || pending == nil {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
		}

		d := IPVerifyData{}
		d.Populate(e)
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Verify Your Identity")
		d.Slug = "/auth/verify-ip"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(ipVerifyTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func IPVerificationVerifyHandler(appStore *store.Store, sessStore *session.Store, mailService *mailer.Service) func(e *server.RequestEvent) error {
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
			if e.Request.Header.Get("HX-Request") != "" {
				return e.String(http.StatusBadRequest, "Please enter the verification code.")
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/auth/verify-ip"))
		}

		ctx := e.Request.Context()

		storedCode, err := appStore.Security.GetIPVerificationCode(ctx, pending.UserID, pending.IP)
		if err != nil || storedCode == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				return e.String(http.StatusBadRequest, "No verification code found. Please request a new one.")
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/auth/verify-ip"))
		}

		if subtle.ConstantTimeCompare([]byte(hashCode(code)), []byte(storedCode.CodeHash)) != 1 {
			pending.FailCount++
			if pending.FailCount >= 5 {
				clearPendingAuthCookie(e)
				if e.Request.Header.Get("HX-Request") != "" {
					e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/login"))
					return e.HTML(http.StatusNoContent, "")
				}
				return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
			}
			_ = setPendingAuthCookie(e, *pending)
			if e.Request.Header.Get("HX-Request") != "" {
				return e.String(http.StatusBadRequest, "Invalid code. Please try again.")
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/auth/verify-ip"))
		}

		_ = appStore.Security.MarkIPVerificationCodeUsed(ctx, storedCode.ID)
		_ = appStore.Security.VerifyKnownIP(ctx, pending.UserID, pending.IP)

		return completeChallenge(e, appStore, sessStore, pending, "ip")
	}
}

func IPVerificationResendHandler(appStore *store.Store, mailService *mailer.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		pending, err := readPendingAuth(e)
		if err != nil || pending == nil {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
		}

		if pending.ResendCount >= 3 {
			if e.Request.Header.Get("HX-Request") != "" {
				return e.String(http.StatusTooManyRequests, "Too many resend attempts. Please try logging in again.")
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/auth/verify-ip"))
		}

		pending.ResendCount++
		_ = setPendingAuthCookie(e, *pending)

		ctx := e.Request.Context()
		sendIPVerificationEmail(ctx, appStore, mailService, pending.UserID, pending.IP)

		if e.Request.Header.Get("HX-Request") != "" {
			return e.String(http.StatusOK, "Verification code sent.")
		}
		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/auth/verify-ip"))
	}
}

func KnownIPDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		ctx := e.Request.Context()
		ipID := e.Request.PathValue("id")

		_ = appStore.Security.DeleteKnownIP(ctx, ipID, userID)

		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/settings/security"))
	}
}
