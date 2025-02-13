package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const newsTemplate = "news.html"

type NewsData struct {
	DefaultData
}

func NewsHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := NewsData{}
		d.Populate(e)
		d.Title = "News"
		d.Categories = allCategories(app)
		html, err := registry.LoadFiles(newsTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
