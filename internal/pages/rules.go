package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const rulesTemplate = "rules.html"

type RulesData struct {
	DefaultData
}

func RulesHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := RulesData{}
		d.Populate(e)
		d.Title = "Rules"
		d.Categories = allCategories(app)
		html, err := registry.LoadFiles(rulesTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
