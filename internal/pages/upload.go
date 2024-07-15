package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const uploadTemplate = "upload.html"

type UploadData struct {
	DefaultData
}

func UploadHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := UploadData{}
		d.Title = "Upload A Schematic"
		d.Categories = allCategories(app)
		err := c.Render(http.StatusOK, uploadTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
