package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const newsTemplate = "news.html"

type NewsData struct {
	DefaultData
}

func NewsHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := NewsData{}
		d.Title = "News"
		err := c.Render(http.StatusOK, newsTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
