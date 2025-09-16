package pages

import (
	"createmod/internal/cache"
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

func APIDocsHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := APIDocsData{}
		d.Populate(e)
		d.Title = "API Documentation"
		d.Description = "Developer documentation for createmod.com API"
		d.Slug = "/api"
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(apiDocsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
