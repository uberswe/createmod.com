package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"github.com/sym01/htmlsanitizer"
	"html/template"
	"net/http"
	"time"
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

// NewsPostHandler renders an individual news post using the record ID in the route.
// For now, the route variable {slug} is treated as a PocketBase record ID because
// the news collection doesn't define a slug field in migrations.
func NewsPostHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := NewsPostData{}
		d.Populate(e)
		d.Categories = allCategories(app, cacheService)

		id := e.Request.PathValue("slug")
		if id != "" {
			coll, err := app.FindCollectionByNameOrId("news")
			if err == nil && coll != nil {
				rec, err := app.FindRecordById(coll.Id, id)
				if err == nil && rec != nil && rec.Id != "" {
					// Populate display fields
					title := rec.GetString("title")
					if title == "" {
						title = "News"
					}
					d.Title = title
					d.Description = rec.GetString("excerpt")
					d.Slug = "/news/" + rec.Id
					// prefer postdate, else updated, else created
					pd := rec.GetDateTime("postdate")
					when := time.Now().UTC()
					if !pd.IsZero() {
						when = pd.Time()
					} else if !rec.GetDateTime("updated").IsZero() {
						when = rec.GetDateTime("updated").Time()
					} else if !rec.GetDateTime("created").IsZero() {
						when = rec.GetDateTime("created").Time()
					}
					d.PostDate = when.Format("January 2, 2006")
					// sanitize content as HTML
					raw := rec.GetString("content")
					if raw != "" {
						sanitizer := htmlsanitizer.NewHTMLSanitizer()
						clean, _ := sanitizer.Sanitize([]byte(raw))
						d.Content = template.HTML(string(clean))
					}
				}
			}
		}

		htmlOut, err := registry.LoadFiles(newsPostTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, htmlOut)
	}
}
