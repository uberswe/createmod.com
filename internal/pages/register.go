package pages

import (
	"createmod/internal/auth"
	"createmod/internal/i18n"
	"createmod/internal/session"
	"createmod/internal/store"
	"net/http"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

const registerTemplate = "./template/register.html"

var registerTemplates = append([]string{
	registerTemplate,
}, commonTemplates...)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type registerData struct {
	DefaultData
	Error    string
	Username string
	Email    string
}

func RegisterHandler(app *pocketbase.PocketBase, registry *template.Registry, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := registerData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "page.register.title")
		d.Description = i18n.T(d.Language, "page.register.description")
		d.Slug = "/register"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		html, err := registry.LoadFiles(registerTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// RegisterPostHandler handles POST /register by creating a new user account.
func RegisterPostHandler(app *pocketbase.PocketBase, appStore *store.Store, sessStore *session.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}

		username := strings.TrimSpace(e.Request.Form.Get("username"))
		email := strings.TrimSpace(e.Request.Form.Get("email"))
		password := e.Request.Form.Get("password")
		terms := e.Request.Form.Get("terms")

		// Validate
		if username == "" || email == "" || password == "" {
			return registerError(e, "All fields are required")
		}
		if terms != "on" && terms != "true" {
			return registerError(e, "You must agree to the Terms of Service")
		}
		if len(password) < 8 {
			return registerError(e, "Password must be at least 8 characters")
		}
		if !emailRegex.MatchString(email) {
			return registerError(e, "Invalid email address")
		}
		if len(username) < 3 || len(username) > 30 {
			return registerError(e, "Username must be 3-30 characters")
		}

		ctx := e.Request.Context()

		// Check uniqueness
		if existing, _ := appStore.Users.GetUserByEmail(ctx, email); existing != nil {
			return registerError(e, "An account with this email already exists")
		}
		if existing, _ := appStore.Users.GetUserByUsername(ctx, username); existing != nil {
			return registerError(e, "This username is already taken")
		}

		// Hash password
		hash, err := auth.HashPassword(password)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to process registration")
		}

		// Create user in PostgreSQL
		newUser := &store.User{
			Email:        email,
			Username:     strings.ToLower(username),
			PasswordHash: hash,
		}
		if err := appStore.Users.CreateUser(ctx, newUser); err != nil {
			app.Logger().Error("registration failed", "error", err)
			return registerError(e, "Registration failed. Please try again.")
		}

		// Sync to PocketBase (transition bridge)
		syncUserToPocketBase(app, newUser)

		// Create session
		token, err := sessStore.Create(ctx, newUser.ID)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to create session")
		}

		secure := e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")
		session.SetCookie(e.Response, token, secure)

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusFound, LangRedirectURL(e, "/"))
	}
}

func registerError(e *core.RequestEvent, msg string) error {
	if e.Request.Header.Get("HX-Request") != "" {
		return e.String(http.StatusBadRequest, msg)
	}
	return e.Redirect(http.StatusFound, LangRedirectURL(e, "/register"))
}
