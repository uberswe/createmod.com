package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const guideTemplate = "./template/dist/guide.html"

type GuideData struct {
	DefaultData
}

func GuideHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := GuideData{}
		d.Populate(e)
		d.Title = "Guide"
		d.Description = "How do you use Create Mod schematic files? This page has a simple guide that should help!"
		d.Slug = "/guide"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(guideTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
