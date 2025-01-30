package migrate

import (
	"createmod/model"
	"createmod/query"
	"errors"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func migrateViews(app *pocketbase.PocketBase, gormdb *gorm.DB, oldUserIDs map[int64]string, oldSchematicIDs map[int64]string) {
	app.Logger().Info("Migrating views.")
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
	postViewRes := make([]*model.QeyKryWEpostView, 0)
	postErr := q.QeyKryWEpostView.FindInBatches(&postViewRes, 5000, func(tx gen.Dao, batch int) error {
		for i := range postViewRes {
			if newSchematicID, ok := oldSchematicIDs[postViewRes[i].ID]; ok {
				filter, err := app.Dao().FindRecordsByFilter(
					schematicViewsCollection.Id,
					"old_id = {:old_id} && type = {:type}",
					"-created",
					1,
					0,
					dbx.Params{
						"old_schematic_id": postViewRes[i].ID,
						"type":             postViewRes[i].Type,
					})
				if !errors.Is(err, gorm.ErrRecordNotFound) && len(filter) != 0 {
					app.Logger().Debug(
						fmt.Sprintf("Rating found or error: %v", err),
						"filter-len", len(filter),
					)
					if err != nil {
						app.Logger().Info(err.Error())
					}
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
		return nil
	})
	if postErr != nil {
		panic(postErr)
	}
}
