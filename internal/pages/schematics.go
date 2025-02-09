package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const schematicsTemplate = "schematics.html"

type SchematicsData struct {
	DefaultData
	Schematics []models.Schematic
}

func SchematicsHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		schematicsCollection, err := app.Dao().FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.Dao().FindRecordsByFilter(
			schematicsCollection.Id,
			"1=1",
			"-created",
			51,
			0)

		d := SchematicsData{
			Schematics: MapResultsToSchematic(app, results),
		}
		d.Populate(c)
		d.Title = "Create Mod Schematics"
		d.Categories = allCategories(app)

		err = c.Render(http.StatusOK, schematicsTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
