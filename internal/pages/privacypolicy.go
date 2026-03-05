package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const privacyPolicyTemplate = "./template/privacy-policy.html"

var privacyPolicyTemplates = append([]string{
	privacyPolicyTemplate,
}, commonTemplates...)

type PrivacyPolicyData struct {
	DefaultData
}

func PrivacyPolicyHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := PrivacyPolicyData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Privacy Policy")
		d.Description = i18n.T(d.Language, "page.privacypolicy.description")
		d.Slug = "/privacy-policy"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(privacyPolicyTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
