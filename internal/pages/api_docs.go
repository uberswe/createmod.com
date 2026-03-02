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

const apiDocsTemplate = "./template/api_docs.html"

var apiDocsTemplates = append([]string{
	apiDocsTemplate,
}, commonTemplates...)

type APIDocsData struct {
	DefaultData
}

func APIDocsHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := APIDocsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "API Documentation")
		d.Description = i18n.T(d.Language, "page.api_docs.description")
		d.Slug = "/api"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(apiDocsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
