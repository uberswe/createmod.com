package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const registerTemplate = "./template/register.html"

var registerTemplates = append([]string{
	registerTemplate,
}, commonTemplates...)

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
		html, err := registry.LoadFiles(registerTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
