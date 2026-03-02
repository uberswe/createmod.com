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

const guideTemplate = "./template/guide.html"

var guideTemplates = append([]string{
	guideTemplate,
}, commonTemplates...)

type GuideData struct {
	DefaultData
}

func GuideHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := GuideData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "page.guide.title")
		d.Description = i18n.T(d.Language, "page.guide.description")
		d.Slug = "/guide"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(guideTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
