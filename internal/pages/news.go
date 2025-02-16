package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const newsTemplate = "./template/dist/news.html"

type NewsData struct {
	DefaultData
}

func NewsHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := NewsData{}
		d.Populate(e)
		d.Title = "News"
		d.Description = "CreateMod.com news features the latest developments on the website."
		d.Slug = "/news"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(newsTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
