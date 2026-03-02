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

const contactTemplate = "./template/contact.html"

var contactTemplates = append([]string{
	contactTemplate,
}, commonTemplates...)

type ContactData struct {
	DefaultData
}

func ContactHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := ContactData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Contact")
		d.Description = i18n.T(d.Language, "page.contact.description")
		d.Slug = "/contact"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(contactTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
