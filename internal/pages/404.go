package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const fourOhFourTemplate = "404.html"

func FourOhFourHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.Render(http.StatusNotFound, fourOhFourTemplate, nil)
	}
}
