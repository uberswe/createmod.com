package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const termsOfServiceTemplate = "./template/dist/terms-of-service.html"

type TermsOfServiceData struct {
	DefaultData
}

func TermsOfServiceHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := TermsOfServiceData{}
		d.Populate(e)
		d.Title = "Terms Of Service"
		d.Categories = allCategories(app)

		html, err := registry.LoadFiles(termsOfServiceTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
