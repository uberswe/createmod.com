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

const privacyPolicyTemplate = "./template/privacy-policy.html"

var privacyPolicyTemplates = append([]string{
	privacyPolicyTemplate,
}, commonTemplates...)

type PrivacyPolicyData struct {
	DefaultData
}

func PrivacyPolicyHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := PrivacyPolicyData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Privacy Policy")
		d.Description = i18n.T(d.Language, "page.privacypolicy.description")
		d.Slug = "/privacy-policy"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(privacyPolicyTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
