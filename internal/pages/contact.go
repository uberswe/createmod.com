package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const contactTemplate = "./template/contact.html"

var contactTemplates = append([]string{
	contactTemplate,
}, commonTemplates...)

type ContactData struct {
	DefaultData
}

func ContactHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := ContactData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Contact"))
		d.Title = i18n.T(d.Language, "Contact")
		d.Description = i18n.T(d.Language, "page.contact.description")
		d.Slug = "/contact"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(contactTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
