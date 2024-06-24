package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const newsPostTemplate = "news_post.html"

type NewsPostData struct {
	DefaultData
}

func NewsPostHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := NewsPostData{}
		d.Title = ""
		err := c.Render(http.StatusOK, newsPostTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
