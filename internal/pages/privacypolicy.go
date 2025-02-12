package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const privacyPolicyTemplate = "privacy-policy.html"

type PrivacyPolicyData struct {
	DefaultData
}

func PrivacyPolicyHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := PrivacyPolicyData{}
		d.Populate(c)
		d.Title = "Privacy Policy"
		d.Categories = allCategories(app)
		err := c.Render(http.StatusOK, privacyPolicyTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
