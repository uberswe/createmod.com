package pages

import (
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"net/http"
	"strings"
	"time"

	"createmod/internal/server"
)

var userSettingsTemplates = append([]string{
	"./template/user-settings.html",
}, commonTemplates...)

type APIKeyItem struct {
	ID      string
	Label   string
	Last8   string
	Created time.Time
}

type UserSettingsData struct {
	DefaultData
	LinkedGithub  bool
	LinkedDiscord bool
	HasPassword   bool
	OAuthError    string
}

func UserSettingsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		d := UserSettingsData{}
		d.Populate(e)
		d.HideOutstream = true
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"))
		d.Title = i18n.T(d.Language, "Settings")
		d.Description = i18n.T(d.Language, "page.usersettings.description")
		d.Slug = "/settings"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		linked, err := appStore.Auth.ListByUser(ctx, userID)
		if err == nil {
			for _, ea := range linked {
				switch ea.Provider {
				case "github":
					d.LinkedGithub = true
				case "discord":
					d.LinkedDiscord = true
				}
			}
		}

		user, err := appStore.Users.GetUserByID(ctx, userID)
		if err == nil && user != nil {
			d.HasPassword = user.PasswordHash != ""
		}

		d.OAuthError = e.Request.URL.Query().Get("oauth_error")

		html, err := registry.LoadFiles(userSettingsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func UnlinkOAuthHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}

		provider := strings.TrimSpace(e.Request.Form.Get("provider"))
		password := strings.TrimSpace(e.Request.Form.Get("password"))

		if provider != "github" && provider != "discord" {
			return e.Redirect(http.StatusSeeOther, "/settings")
		}

		if password == "" {
			return e.Redirect(http.StatusSeeOther, "/settings?oauth_error=password_required")
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		user, err := appStore.Users.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return e.Redirect(http.StatusSeeOther, "/settings?oauth_error=user_not_found")
		}

		matched, _ := auth.CheckPassword(user.PasswordHash, user.OldPassword, password)
		if !matched {
			return e.Redirect(http.StatusSeeOther, "/settings?oauth_error=wrong_password")
		}

		if err := appStore.Auth.DeleteByProvider(ctx, userID, provider); err != nil {
			return e.Redirect(http.StatusSeeOther, "/settings?oauth_error=unlink_failed")
		}

		return e.Redirect(http.StatusSeeOther, "/settings")
	}
}
