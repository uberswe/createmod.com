package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const fourOhFourTemplate = "404.html"

func FourOhFourHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		html, err := registry.LoadFiles(fourOhFourTemplate).Render(nil)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusNotFound, html)
	}
}
