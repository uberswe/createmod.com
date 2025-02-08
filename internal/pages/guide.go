package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const guideTemplate = "guide.html"

type GuideData struct {
	DefaultData
}

func GuideHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := GuideData{}
		d.Populate(c)
		d.Title = "Guide"
		d.Categories = allCategories(app)
		err := c.Render(http.StatusOK, guideTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
