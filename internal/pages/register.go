package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const registerTemplate = "./template/dist/register.html"

type registerData struct {
	DefaultData
}

func RegisterHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := registerData{}
		d.Populate(e)
		d.Title = "Register"
		d.Description = "The CreateMod.com register page."
		d.Slug = "/register"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app)
		html, err := registry.LoadFiles(registerTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
