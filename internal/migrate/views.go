package migrate

import (
	"createmod/query"
	"errors"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"gorm.io/gorm"
	"log"
)

func migrateViews(app *pocketbase.PocketBase, gormdb *gorm.DB, oldUserIDs map[int64]string, oldSchematicIDs map[int64]string) {
	log.Println("Migrating views.")
	// QeyKryWEpost_views
	// id
	// type
	// period
	// count
	q := query.Use(gormdb)
	schematicViewsCollection, err := app.Dao().FindCollectionByNameOrId("schematic_views")
	if err != nil {
		panic(err)
	}

	postViewRes, viewErr := q.QeyKryWEpostView.Find()
	for i := range postViewRes {
		if newSchematicID, ok := oldSchematicIDs[postViewRes[i].ID]; ok {
			filter, err := app.Dao().FindRecordsByFilter(
				schematicViewsCollection.Id,
				"old_schematic_id = {:old_schematic_id} && type = {:type}",
				"-created",
				1,
				0,
				dbx.Params{
					"old_schematic_id": postViewRes[i].ID,
					"type":             postViewRes[i].Type,
				})
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				app.Logger().Debug(
					fmt.Sprintf("Views error: %v", err),
					"filter-len", len(filter),
				)
				continue
			} else if err == nil && len(filter) != 0 {
				app.Logger().Debug(
					fmt.Sprintf("Views found: %v", err),
					"filter-len", len(filter),
				)
				continue
			}

			record := models.NewRecord(schematicViewsCollection)
			record.Set("old_schematic_id", postViewRes[i].ID)
			record.Set("schematic", newSchematicID)
			record.Set("count", postViewRes[i].Count_)
			record.Set("type", postViewRes[i].Type)
			record.Set("period", postViewRes[i].Period)

			if err = app.Dao().SaveRecord(record); err != nil {
				panic(err)
			}
		}
	}
	if viewErr != nil {
		panic(viewErr)
	}
}
