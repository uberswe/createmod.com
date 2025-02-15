package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const loginTemplate = "./template/dist/login.html"

type LoginData struct {
	DefaultData
}

func LoginHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := LoginData{}
		d.Populate(e)
		d.Title = "Login"
		d.Description = "Login to CreateMod.com to upload your schematics or to rate and comment on other builds."
		d.Slug = "/login"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		html, err := registry.LoadFiles(loginTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
