package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const guideTemplate = "./template/dist/guide.html"

type GuideData struct {
	DefaultData
}

func GuideHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := GuideData{}
		d.Populate(e)
		d.Title = "Guide"
		d.Categories = allCategories(app)
		html, err := registry.LoadFiles(guideTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
