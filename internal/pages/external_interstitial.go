package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"net/url"
)

const externalInterstitialTemplate = "./template/external_interstitial.html"

var externalInterstitialTemplates = append([]string{
	externalInterstitialTemplate,
}, commonTemplates...)

type ExternalInterstitialData struct {
	DefaultData
	Target string
}

func ExternalLinkInterstitialHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		raw := e.Request.URL.Query().Get("url")
		if raw == "" {
			return e.String(http.StatusBadRequest, "missing url parameter")
		}
		// basic safety: only allow http/https and valid URL
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return e.String(http.StatusBadRequest, "invalid url parameter")
		}
		// reconstruct absolute/escaped target
		target := u.String()

		d := ExternalInterstitialData{}
		d.Populate(e)
		d.Title = "You are leaving createmod.com"
		d.Description = "External link warning"
		d.Slug = "/out"
		d.Categories = allCategories(app, cacheService)
		d.Target = target

		// If a guide id is provided, increment its views best-effort
		if guideID := e.Request.URL.Query().Get("guide"); guideID != "" {
			if coll, err := app.FindCollectionByNameOrId("guides"); err == nil && coll != nil {
				if rec, err := app.FindRecordById(coll.Id, guideID); err == nil && rec != nil {
					current := rec.GetInt("views")
					rec.Set("views", current+1)
					if err := app.Save(rec); err != nil {
						app.Logger().Warn("guides: failed to increment views", "error", err)
					}
				}
			}
		}

		html, err := registry.LoadFiles(externalInterstitialTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
