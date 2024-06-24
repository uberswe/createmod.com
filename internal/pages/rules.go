package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const rulesTemplate = "rules.html"

type RulesData struct {
	DefaultData
}

func RulesHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := RulesData{}
		d.Title = "Rules"
		err := c.Render(http.StatusOK, rulesTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
