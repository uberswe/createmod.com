package pages

import (
	"createmod/internal/models"
	"github.com/pocketbase/pocketbase"
)

type DefaultData struct {
	Title       string
	SubCategory string
	Categories  []models.SchematicCategory
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
