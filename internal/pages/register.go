package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const registerTemplate = "register.html"

type registerData struct {
	DefaultData
}

func RegisterHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := registerData{}
		d.Title = "Register"
		d.Categories = allCategories(app)
		err := c.Render(http.StatusOK, registerTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
