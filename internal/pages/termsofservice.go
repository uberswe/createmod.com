package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const termsOfServiceTemplate = "terms-of-service.html"

type TermsOfServiceData struct {
	DefaultData
}

func TermsOfServiceHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := TermsOfServiceData{}
		d.Populate(c)
		d.Title = "Terms Of Service"
		d.Categories = allCategories(app)
		err := c.Render(http.StatusOK, termsOfServiceTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
