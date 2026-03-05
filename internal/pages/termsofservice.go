package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const termsOfServiceTemplate = "./template/terms-of-service.html"

var termsOfServiceTemplates = append([]string{
	termsOfServiceTemplate,
}, commonTemplates...)

type TermsOfServiceData struct {
	DefaultData
}

func TermsOfServiceHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := TermsOfServiceData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "page.termsofservice.title")
		d.Description = i18n.T(d.Language, "page.termsofservice.description")
		d.Slug = "/terms-of-service"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(termsOfServiceTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
