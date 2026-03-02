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

const termsOfServiceTemplate = "./template/terms-of-service.html"

var termsOfServiceTemplates = append([]string{
	termsOfServiceTemplate,
}, commonTemplates...)

type TermsOfServiceData struct {
	DefaultData
}

func TermsOfServiceHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := TermsOfServiceData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "page.termsofservice.title")
		d.Description = i18n.T(d.Language, "page.termsofservice.description")
		d.Slug = "/terms-of-service"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		html, err := registry.LoadFiles(termsOfServiceTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
