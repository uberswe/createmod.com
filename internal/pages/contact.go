package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const contactTemplate = "./template/dist/contact.html"

type ContactData struct {
	DefaultData
}

func ContactHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := ContactData{}
		d.Populate(e)
		d.Title = "Contact"
		d.Description = "Contact the CreateMod.com maintainers in case you have a question or suggestion."
		d.Slug = "/contact"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(contactTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
