package pages

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const privacyPolicyTemplate = "./template/dist/privacy-policy.html"

type PrivacyPolicyData struct {
	DefaultData
}

func PrivacyPolicyHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := PrivacyPolicyData{}
		d.Populate(e)
		d.Title = "Privacy Policy"
		d.Categories = allCategories(app)
		html, err := registry.LoadFiles(privacyPolicyTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
