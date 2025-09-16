package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"time"
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

		// Load latest news posts
		recs, err := app.FindRecordsByFilter("news", "1=1", "-postdate", 50, 0)
		if err == nil {
			posts := make([]models.NewsPostListItem, 0, len(recs))
			for i := range recs {
				when := time.Now().UTC()
				if dt := recs[i].GetDateTime("postdate"); !dt.IsZero() {
					when = dt.Time()
				} else if dt := recs[i].GetDateTime("updated"); !dt.IsZero() {
					when = dt.Time()
				} else if dt := recs[i].GetDateTime("created"); !dt.IsZero() {
					when = dt.Time()
				}
				posts = append(posts, models.NewsPostListItem{
					ID:       recs[i].Id,
					Title:    recs[i].GetString("title"),
					Excerpt:  recs[i].GetString("excerpt"),
					URL:      "/news/" + recs[i].Id,
					PostDate: when,
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
