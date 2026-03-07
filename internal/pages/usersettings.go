package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
	"time"
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
}

func UserSettingsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		d := UserSettingsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Settings")
		d.Description = i18n.T(d.Language, "page.usersettings.description")
		d.Slug = "/settings"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(userSettingsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
