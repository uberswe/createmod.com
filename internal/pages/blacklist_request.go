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

var blacklistRequestTemplates = append([]string{
	"./template/blacklist_request.html",
}, commonTemplates...)

type BlacklistRequestData struct {
	DefaultData
}

// BlacklistRequestHandler renders a simple page that lets authenticated users
// submit a blacklist request using the existing /reports endpoint.
func BlacklistRequestHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Require login so we can associate reporter + user id
		if ok, err := requireAuth(e); !ok {
			return err
		}
		d := BlacklistRequestData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Request schematic blacklisting")
		d.Description = i18n.T(d.Language, "page.blacklist.description")
		d.Slug = "/settings/blacklist"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(blacklistRequestTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
