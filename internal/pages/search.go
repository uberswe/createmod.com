package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const searchTemplate = "search.html"

type SearchData struct {
	DefaultData
	Schematics []models.Schematic
}

func SearchHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		schematicsCollection, err := app.Dao().FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.Dao().FindRecordsByFilter(
			schematicsCollection.Id,
			"1=1",
			"-created",
			30,
			0)

		d := SearchData{
			Schematics: mapResultsToSchematic(app, results),
		}
		d.Title = "Search"

		err = c.Render(http.StatusOK, searchTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
