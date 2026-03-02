package pages

import (
	"createmod/content"
	"createmod/internal/cache"
	"createmod/internal/news"
	"createmod/internal/store"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"html/template"
	"net/http"
)

const newsPostTemplate = "./template/news_post.html"

var newsPostTemplates = append([]string{
	newsPostTemplate,
}, commonTemplates...)

type NewsPostData struct {
	DefaultData
	PostDate string
	Content  template.HTML
}

func NewsPostHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := NewsPostData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		slug := e.Request.PathValue("slug")
		if slug != "" {
			post, err := news.LoadBySlug(content.NewsFS, "news", slug)
			if err == nil && post != nil {
				d.Title = post.Title
				d.Description = post.Excerpt
				d.Slug = post.URL
				d.PostDate = post.Date.Format("January 2, 2006")
				d.Content = post.Body
			}
		}

		htmlOut, err := registry.LoadFiles(newsPostTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, htmlOut)
	}
}
