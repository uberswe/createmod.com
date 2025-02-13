package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const contactTemplate = "./template/dist/contact.html"

type ContactData struct {
	DefaultData
}

func ContactHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := ContactData{}
		d.Populate(e)
		d.Title = "Contact"
		d.Categories = allCategories(app)
		html, err := registry.LoadFiles(contactTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
