package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const passwordResetTemplate = "password-reset.html"

type passwordResetData struct {
	DefaultData
}

func PasswordResetHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := passwordResetData{}
		d.Populate(e)
		d.Title = "Reset Password"
		d.Categories = allCategories(app)
		html, err := registry.LoadFiles(passwordResetTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
