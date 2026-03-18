package pages

import (
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const loginTemplate = "./template/login.html"

var loginTemplates = append([]string{
	loginTemplate,
}, commonTemplates...)

type LoginData struct {
	DefaultData
}

func LoginHandler(registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := LoginData{}
		d.Populate(e)
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Login")
		d.Description = i18n.T(d.Language, "page.login.description")
		d.Slug = "/login"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		html, err := registry.LoadFiles(loginTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
