package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
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

func UserSettingsHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Require auth
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}

		d := UserSettingsData{}
		d.Populate(e)
		d.Title = "Settings"
		d.Description = "The user settings page."
		d.Slug = "/settings"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)

		html, err := registry.LoadFiles(userSettingsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
