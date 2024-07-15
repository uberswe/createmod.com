package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const profileTemplate = "profile.html"

type ProfileData struct {
	DefaultData
}

func ProfileHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := ProfileData{}
		d.Title = ""
		d.Categories = allCategories(app)
		err := c.Render(http.StatusOK, profileTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
