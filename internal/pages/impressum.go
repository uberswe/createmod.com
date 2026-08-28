package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
)

const impressumTemplate = "./template/impressum.html"

var impressumTemplates = append([]string{
	impressumTemplate,
}, commonTemplates...)

type ImpressumData struct {
	DefaultData
}

// ImpressumHandler renders the legal notice (Impressum) required under German
// law (§ 5 DDG). Linked from the footer on every page. (#1646)
func ImpressumHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := ImpressumData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "page.impressum.title"))
		d.Title = i18n.T(d.Language, "page.impressum.title")
		d.Description = i18n.T(d.Language, "page.impressum.description")
		d.Slug = "/impressum"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(impressumTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
