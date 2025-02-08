package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
)

type DefaultData struct {
	IsAuthenticated bool
	Title           string
	SubCategory     string
	Categories      []models.SchematicCategory
}

func (d *DefaultData) Populate(c echo.Context) {
	user := c.Get(apis.ContextAuthRecordKey)
	if user != nil {
		d.IsAuthenticated = true
	}
}

func allCategories(app *pocketbase.PocketBase) []models.SchematicCategory {
	categoriesCollection, err := app.Dao().FindCollectionByNameOrId("schematic_categories")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByFilter(categoriesCollection.Id, "1=1", "+name", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToCategories(records)
}
