package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const passwordResetTemplate = "./template/dist/password-reset.html"

type passwordResetData struct {
	DefaultData
}

func PasswordResetHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := passwordResetData{}
		d.Populate(e)
		d.Title = "Reset Password"
		d.Description = "The CreateMod.com reset password page."
		d.Slug = "/reset-password"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app)
		html, err := registry.LoadFiles(passwordResetTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
