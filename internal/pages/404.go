package pages

import (
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const fourOhFourTemplate = "./template/404.html"

var fourOhFourTemplates = append([]string{
	fourOhFourTemplate,
}, commonTemplates...)

func FourOhFourHandler(registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
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
