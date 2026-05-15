package pages

import (
	"createmod/internal/server"
	"net/http"
)

const styleguideTemplate = "./template/styleguide.html"

var styleguideTemplates = append([]string{
	styleguideTemplate,
}, commonTemplates...)

type StyleguideData struct {
	DefaultData
}

func StyleguideHandler(registry *server.Registry) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := StyleguideData{}
		d.Populate(e)
		d.Title = "Style Guide"
		d.NoIndex = true
		html, err := registry.LoadFiles(styleguideTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
