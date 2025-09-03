package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const newsPostTemplate = "./template/news_post.html"

var newsPostTemplates = append([]string{
	newsPostTemplate,
}, commonTemplates...)

type NewsPostData struct {
	DefaultData
}

func NewsPostHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := NewsPostData{}
		d.Populate(e)
		d.Title = ""
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(newsPostTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
