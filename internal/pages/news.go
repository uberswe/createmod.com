package pages

import (
	"createmod/content"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/news"
	"createmod/internal/store"
	"html/template"
	"net/http"
	"strings"

	"createmod/internal/server"
)


const newsTemplate = "./template/news.html"

var newsTemplates = append([]string{
	newsTemplate,
}, commonTemplates...)

type NewsData struct {
	DefaultData
	Posts []models.NewsPostListItem
}

func NewsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := NewsData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "News"))
		d.Title = i18n.T(d.Language, "News")
		d.Description = i18n.T(d.Language, "page.news.description")
		d.Slug = "/news"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		// Load news from embedded markdown files
		all, err := news.LoadAll(content.NewsFS, "news")
		if err == nil {
			posts := make([]models.NewsPostListItem, 0, len(all))
			for _, p := range all {
				posts = append(posts, models.NewsPostListItem{
					ID:             p.Slug,
					Title:          p.Title,
					Excerpt:        p.Excerpt,
					FirstParagraph: extractFirstParagraph(p.Body),
					Image:          p.Image,
					URL:            p.URL,
					PostDate:       p.Date,
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

// extractFirstParagraph returns the content of the first <p>...</p> tag in
// rendered HTML. Returns empty if no paragraph is found.
func extractFirstParagraph(body template.HTML) template.HTML {
	s := string(body)
	start := strings.Index(s, "<p>")
	if start < 0 {
		return ""
	}
	end := strings.Index(s[start:], "</p>")
	if end < 0 {
		return ""
	}
	return template.HTML(s[start : start+end+4])
}

