package pages

import (
	"createmod/content"
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/news"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const newsTemplate = "./template/news.html"

var newsTemplates = append([]string{
	newsTemplate,
}, commonTemplates...)

type NewsData struct {
	DefaultData
	Posts []models.NewsPostListItem
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

		// Load news from embedded markdown files
		all, err := news.LoadAll(content.NewsFS, "news")
		if err == nil {
			posts := make([]models.NewsPostListItem, 0, len(all))
			for _, p := range all {
				posts = append(posts, models.NewsPostListItem{
					ID:       p.Slug,
					Title:    p.Title,
					Excerpt:  p.Excerpt,
					URL:      p.URL,
					PostDate: p.Date,
				})
			}
			d.Posts = posts
		}

		html, err := registry.LoadFiles(newsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
