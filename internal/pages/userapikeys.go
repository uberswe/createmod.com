package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
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

func UserAPIKeysHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)

		d := UserAPIKeysData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "API Keys")
		d.Description = i18n.T(d.Language, "page.userapikeys.description")
		d.Slug = "/settings/api-keys"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		// Load user's API keys
		ctx := e.Request.Context()
		keys, err := appStore.APIKeys.ListByUser(ctx, userID)
		if err == nil {
			items := make([]APIKeyItem, 0, len(keys))
			for _, k := range keys {
				items = append(items, APIKeyItem{
					ID:      k.ID,
					Label:   k.Label,
					Last8:   k.Last8,
					Created: k.Created,
				})
			}
			d.APIKeys = items
		}

		// One-time new API key display
		if key, ok := cacheService.GetString("apikey:new:" + userID); ok && key != "" {
			d.NewAPIKey = key
			cacheService.Delete("apikey:new:" + userID)
		}

		html, err := registry.LoadFiles(userAPIKeysTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
