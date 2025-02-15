package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

var userSettingsTemplates = []string{
	"./template/dist/user-settings.html",
}

type UserSettingsData struct {
	DefaultData
}

func UserSettingsHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := UserSettingsData{}
		d.Populate(e)
		d.Title = "Settings"
		d.Categories = allCategories(app)

		html, err := registry.LoadFiles(userSettingsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
