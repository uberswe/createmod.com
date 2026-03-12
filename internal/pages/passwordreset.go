package pages

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"createmod/internal/auth"
	"createmod/internal/i18n"
	"createmod/internal/mailer"
	"createmod/internal/session"
	"createmod/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"createmod/internal/server"
)

// pool is set by SetPasswordResetPool so handlers can access it.
var resetPool *pgxpool.Pool

// SetPasswordResetPool stores a reference to the pgx pool for password reset queries.
func SetPasswordResetPool(p *pgxpool.Pool) {
	resetPool = p
}

const passwordResetTemplate = "./template/password-reset.html"
const passwordResetConfirmTemplate = "./template/password-reset-confirm.html"

var passwordResetTemplates = append([]string{
	passwordResetTemplate,
}, commonTemplates...)

var passwordResetConfirmTemplates = append([]string{
	passwordResetConfirmTemplate,
}, commonTemplates...)

type passwordResetData struct {
	DefaultData
	Success bool
	Error   string
}

type passwordResetConfirmData struct {
	DefaultData
	Token   string
	Error   string
	Success bool
}

func PasswordResetHandler(registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := passwordResetData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "page.passwordreset.title")
		d.Description = i18n.T(d.Language, "page.passwordreset.description")
		d.Slug = "/reset-password"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		html, err := registry.LoadFiles(passwordResetTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// PasswordResetPostHandler handles POST /reset-password.
// It sends a password reset email if the user exists.
func PasswordResetPostHandler(mailService *mailer.Service, registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		email := strings.TrimSpace(e.Request.Form.Get("email"))

		// Always show success to avoid email enumeration
		d := passwordResetData{Success: true}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "page.passwordreset.title")
		d.Slug = "/reset-password"

		if email == "" {
			d.Success = false
			d.Error = "Please enter your email address"
			html, err := registry.LoadFiles(passwordResetTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusOK, html)
		}

		// Look up user
		ctx := e.Request.Context()
		user, _ := appStore.Users.GetUserByEmail(ctx, email)
		if user != nil && user.Deleted == nil {
			// Generate token
			tokenBytes := make([]byte, 32)
			if _, err := rand.Read(tokenBytes); err == nil {
				rawToken := hex.EncodeToString(tokenBytes)
				tokenHash := hashToken(rawToken)

				// Store in database
				_, err := resetPool.Exec(ctx,
					`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created)
					 VALUES (gen_random_uuid(), $1, $2, $3, NOW())`,
					user.ID, tokenHash, time.Now().Add(1*time.Hour),
				)
				if err != nil {
					slog.Error("failed to create password reset token", "error", err)
				} else {
					// Send email
					baseURL := "https://createmod.com"
					if devURL := strings.TrimSpace(fmt.Sprintf("%s", e.Request.Host)); strings.Contains(devURL, "localhost") || strings.Contains(devURL, "127.0.0.1") {
						scheme := "http"
						if e.Request.TLS != nil {
							scheme = "https"
						}
						baseURL = scheme + "://" + devURL
					}

					resetURL := baseURL + "/reset-password/" + rawToken

					message := &mailer.Message{
						From: mail.Address{
							Address: mailService.SenderAddress,
							Name:    mailService.SenderName,
						},
						To:      []mail.Address{{Address: user.Email}},
						Subject: "Password Reset - CreateMod.com",
						HTML: fmt.Sprintf(`<p>You requested a password reset for your CreateMod.com account.</p>
<p><a href="%s">Click here to reset your password</a></p>
<p>This link will expire in 1 hour.</p>
<p>If you did not request this, please ignore this email.</p>`, resetURL),
					}
					if err := mailService.Send(message); err != nil {
						slog.Error("failed to send password reset email", "error", err)
					}
				}
			}
		}

		html, err := registry.LoadFiles(passwordResetTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// PasswordResetConfirmHandler renders the new password form.
func PasswordResetConfirmHandler(registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		token := e.Request.PathValue("token")
		d := passwordResetConfirmData{Token: token}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Set New Password")
		d.Slug = "/reset-password/" + token
		html, err := registry.LoadFiles(passwordResetConfirmTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// PasswordResetConfirmPostHandler handles POST /reset-password/{token}.
func PasswordResetConfirmPostHandler(registry *server.Registry, appStore *store.Store, sessStore *session.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		token := e.Request.PathValue("token")
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		password := e.Request.Form.Get("password")

		d := passwordResetConfirmData{Token: token}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Set New Password")
		d.Slug = "/reset-password/" + token

		if password == "" || len(password) < 8 {
			d.Error = "Password must be at least 8 characters"
			html, err := registry.LoadFiles(passwordResetConfirmTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusOK, html)
		}

		ctx := e.Request.Context()
		tokenHash := hashToken(token)

		// Look up token
		var userID string
		var tokenID string
		err := resetPool.QueryRow(ctx,
			`SELECT id, user_id FROM password_reset_tokens
			 WHERE token_hash = $1 AND expires_at > NOW()
			 LIMIT 1`,
			tokenHash,
		).Scan(&tokenID, &userID)

		if err != nil || userID == "" {
			d.Error = "Invalid or expired reset link. Please request a new one."
			html, err := registry.LoadFiles(passwordResetConfirmTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusOK, html)
		}

		// Hash new password
		hash, err := auth.HashPassword(password)
		if err != nil {
			d.Error = "Failed to process password. Please try again."
			html, err := registry.LoadFiles(passwordResetConfirmTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusOK, html)
		}

		// Update password
		if err := appStore.Users.UpdateUserPassword(ctx, userID, hash); err != nil {
			d.Error = "Failed to update password. Please try again."
			html, err := registry.LoadFiles(passwordResetConfirmTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusOK, html)
		}

		// Delete the used token
		_, _ = resetPool.Exec(ctx,
			`DELETE FROM password_reset_tokens WHERE id = $1`, tokenID)

		// Invalidate all existing sessions for the user
		_ = sessStore.DeleteUserSessions(ctx, userID)

		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
	}
}

// hashToken creates a SHA-256 hash of the raw token for storage.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
