package pages

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/session"
	"createmod/internal/store"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"net/http"

	"createmod/internal/server"
)

func maybeCreateSessionOrChallenge(e *server.RequestEvent, appStore *store.Store, sessStore *session.Store, mailService *mailer.Service, userID, returnTo string) error {
	ctx := e.Request.Context()

	settings, _ := appStore.Security.GetSecuritySettings(ctx, userID)
	var needs []string

	if settings != nil && settings.TOTPEnabled {
		totp, _ := appStore.Security.GetTOTP(ctx, userID)
		if totp != nil && totp.Enabled {
			needs = append(needs, "totp")
		}
	}

	if settings != nil && settings.NewIPVerification {
		clientIP := e.RealIP()
		known, _ := appStore.Security.GetKnownIP(ctx, userID, clientIP)
		if known == nil || !known.Verified {
			needs = append(needs, "ip")
			_ = appStore.Security.UpsertKnownIP(ctx, &store.KnownIP{
				UserID:    userID,
				IPAddress: clientIP,
				UserAgent: e.Request.UserAgent(),
				Verified:  false,
			})
		}
	}

	if len(needs) > 0 {
		if returnTo == "" {
			returnTo = "/"
		}
		pending := pendingAuth{
			UserID:   userID,
			IP:       e.RealIP(),
			ReturnTo: returnTo,
			Needs:    strings.Join(needs, ","),
		}
		if err := setPendingAuthCookie(e, pending); err != nil {
			return e.String(http.StatusInternalServerError, "security check failed")
		}

		if contains(needs, "ip") {
			go sendIPVerificationEmail(appStore, mailService, userID, e.RealIP())
		}

		first := needs[0]
		redirectURL := challengeURL(first)

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, redirectURL))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusFound, LangRedirectURL(e, redirectURL))
	}

	token, err := sessStore.Create(ctx, userID)
	if err != nil {
		return e.String(http.StatusInternalServerError, "failed to create session")
	}

	secure := e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")
	session.SetCookie(e.Response, token, secure)

	if returnTo == "" {
		returnTo = "/"
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo))
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusFound, LangRedirectURL(e, returnTo))
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func sendIPVerificationEmail(appStore *store.Store, mailService *mailer.Service, userID, ipAddress string) {
	if mailService == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := appStore.Users.GetUserByID(ctx, userID)
	if err != nil || user == nil || user.Email == "" {
		slog.Warn("ip verification: cannot find user email", "userID", userID, "error", err)
		return
	}

	raw, codeHash, err := generateVerificationCode()
	if err != nil {
		slog.Error("ip verification: failed to generate code", "error", err)
		return
	}

	err = appStore.Security.CreateIPVerificationCode(ctx, &store.IPVerificationCode{
		UserID:    userID,
		IPAddress: ipAddress,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	})
	if err != nil {
		slog.Error("ip verification: failed to store code", "error", err)
		return
	}

	body := "Your verification code is: <strong>" + raw + "</strong>" +
		"<br><br>This code expires in 15 minutes." +
		"<br><br>If you didn't try to log in, please change your password immediately."

	msg := &mailer.Message{
		From:    mailService.DefaultFrom(),
		To:      []mail.Address{{Address: user.Email}},
		Subject: "CreateMod.com - Verify your login",
		HTML:    mailer.EmailHTMLRaw("Login Verification", "", "", "", body),
	}
	if err := mailService.Send(msg); err != nil {
		slog.Error("ip verification: failed to send email", "error", err)
	}
}
