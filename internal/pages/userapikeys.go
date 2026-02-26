package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

var userAPIKeysTemplates = append([]string{
	"./template/user-api-keys.html",
}, commonTemplates...)

type UserAPIKeysData struct {
	DefaultData
	APIKeys   []APIKeyItem
	NewAPIKey string
}

func UserAPIKeysHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Require auth
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}

		d := UserAPIKeysData{}
		d.Populate(e)
		d.Title = "API Keys"
		d.Description = "Manage your API keys."
		d.Slug = "/settings/api-keys"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)

		// Load user's API keys (best-effort)
		if coll, err := app.FindCollectionByNameOrId("api_keys"); err == nil && coll != nil {
			recs, _ := app.FindRecordsByFilter(coll.Id, "user = {:u}", "-created", 200, 0, dbx.Params{"u": e.Auth.Id})
			items := make([]APIKeyItem, 0, len(recs))
			for _, r := range recs {
				items = append(items, APIKeyItem{
					ID:      r.Id,
					Label:   r.GetString("label"),
					Last8:   r.GetString("last8"),
					Created: r.GetDateTime("created").Time(),
				})
			}
			d.APIKeys = items
		}

		// One-time new API key display
		if key, ok := cacheService.GetString("apikey:new:" + e.Auth.Id); ok && key != "" {
			d.NewAPIKey = key
			cacheService.Delete("apikey:new:" + e.Auth.Id)
		}

		html, err := registry.LoadFiles(userAPIKeysTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
