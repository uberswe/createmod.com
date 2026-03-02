package pages

import (
	"createmod/internal/i18n"
	"createmod/internal/store"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const fourOhFourTemplate = "./template/404.html"

var fourOhFourTemplates = append([]string{
	fourOhFourTemplate,
}, commonTemplates...)

func FourOhFourHandler(app *pocketbase.PocketBase, registry *template.Registry, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := DefaultData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Page Not Found")
		d.Description = i18n.T(d.Language, "page.404.description")
		d.Slug = "/404"
		html, err := registry.LoadFiles(fourOhFourTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusNotFound, html)
	}
}
