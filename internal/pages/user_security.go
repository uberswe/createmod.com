package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
)

var userSecurityTemplates = append([]string{
	"./template/user-security.html",
}, commonTemplates...)

type UserSecurityData struct {
	DefaultData
	Settings       *store.SecuritySettings
	TOTPEnabled    bool
	HasPasskeys    bool
	Passkeys       []store.Passkey
	IPVerification bool
}

func UserSecurityHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		d := UserSecurityData{}
		d.Populate(e)
		d.SettingsPage = "security"
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Security")
		d.Slug = "/settings/security"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Security"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		settings, err := appStore.Security.GetSecuritySettings(ctx, userID)
		if err == nil && settings != nil {
			d.Settings = settings
			d.TOTPEnabled = settings.TOTPEnabled
			d.IPVerification = settings.NewIPVerification
		}

		passkeys, err := appStore.Security.ListPasskeys(ctx, userID)
		if err == nil {
			d.Passkeys = passkeys
			d.HasPasskeys = len(passkeys) > 0
		}

		html, err := registry.LoadFiles(userSecurityTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
