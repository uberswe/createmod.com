package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const aboutTemplate = "about.html"

type AboutData struct {
	DefaultData
}

func AboutHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := AboutData{}
		d.Title = "About"
		d.Categories = allCategories(app)
		err := c.Render(http.StatusOK, aboutTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
