package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const termsOfServiceTemplate = "./template/dist/terms-of-service.html"

type TermsOfServiceData struct {
	DefaultData
}

func TermsOfServiceHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := TermsOfServiceData{}
		d.Populate(e)
		d.Title = "Terms Of Service"
		d.Description = "The CreateMod.com terms of service."
		d.Slug = "/terms-of-service"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)

		html, err := registry.LoadFiles(termsOfServiceTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
