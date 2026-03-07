package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const rulesTemplate = "./template/rules.html"

var rulesTemplates = append([]string{
	rulesTemplate,
}, commonTemplates...)

type RulesData struct {
	DefaultData
}

func RulesHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := RulesData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Rules")
		d.Description = i18n.T(d.Language, "page.rules.description")
		d.Slug = "/rules"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(rulesTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
