package pages

import (
	"createmod/content"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/news"
	"createmod/internal/store"
	"createmod/internal/server"
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
	Image    string
	Content  template.HTML
}

func NewsPostHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := NewsPostData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		slug := e.Request.PathValue("slug")
		if slug != "" {
			post, err := news.LoadBySlug(content.NewsFS, "news", slug)
			if err == nil && post != nil {
				d.Title = post.Title
				d.Description = truncateMetaDescription(post.Excerpt)
				d.OGType = "article"
				d.Slug = post.URL
				d.PostDate = post.Date.Format("January 2, 2006")
				d.Content = post.Body
				d.Image = post.Image
				d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "News"), "/news", d.Title)
			}
		}

		htmlOut, err := registry.LoadFiles(newsPostTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, htmlOut)
	}
}
