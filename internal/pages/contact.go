package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const contactTemplate = "contact.html"

type ContactData struct {
	DefaultData
}

func ContactHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := ContactData{}
		d.Title = "Contact"
		err := c.Render(http.StatusOK, contactTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
