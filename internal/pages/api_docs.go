package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const apiDocsTemplate = "./template/api_docs.html"

var apiDocsTemplates = append([]string{
	apiDocsTemplate,
}, commonTemplates...)

type APIDocsData struct {
	DefaultData
}

func APIDocsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := APIDocsData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "API Documentation"))
		d.Title = i18n.T(d.Language, "API Documentation")
		d.Description = i18n.T(d.Language, "page.api_docs.description")
		d.Slug = "/api"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(apiDocsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
