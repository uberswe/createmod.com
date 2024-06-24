package migrate

import (
	"createmod/query"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"gorm.io/gorm"
)

func migrateViews(app *pocketbase.PocketBase, gormdb *gorm.DB, oldUserIDs map[int64]string, oldSchematicIDs map[int64]string) {
	// TODO check if view exists, if it does we skip
	// QeyKryWEpost_views
	// id
	// type
	// period
	// count
	q := query.Use(gormdb)
	postViewRes, postErr := q.QeyKryWEpostView.Find()
	if postErr != nil {
		panic(postErr)
	}

	schematicViewsCollection, err := app.Dao().FindCollectionByNameOrId("schematic_views")
	if err != nil {
		panic(err)
	}

	for i := range postViewRes {
		if newSchematicID, ok := oldSchematicIDs[postViewRes[i].ID]; ok {
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
}
