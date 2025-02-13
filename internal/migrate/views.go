package migrate

import (
	"createmod/model"
	"createmod/query"
	"errors"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"gorm.io/gen"
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
	schematicViewsCollection, err := app.FindCollectionByNameOrId("schematic_views")
	if err != nil {
		panic(err)
	}
	types := map[int32]string{
		4: "total",
		3: "yearly",
		2: "monthly",
		1: "weekly",
		0: "daily",
	}

	updated := 0
	for i, typeDesc := range types {
		log.Printf("View type %d - %s fetching...\n", i, typeDesc)
		var postViewRes []*model.QeyKryWEpostView
		viewErr := q.QeyKryWEpostView.Where(q.QeyKryWEpostView.Type.Eq(i)).FindInBatches(&postViewRes, 100*1000, func(tx gen.Dao, batch int) error {
			log.Printf("View type %d - %s batch %d\n", i, typeDesc, batch)
			for i := range postViewRes {
				if newSchematicID, ok := oldSchematicIDs[postViewRes[i].ID]; ok {
					filter, err := app.FindRecordsByFilter(
						schematicViewsCollection.Id,
						"old_schematic_id = {:old_schematic_id} && type = {:type} && period = {:period}",
						"-created",
						1,
						0,
						dbx.Params{
							"old_schematic_id": postViewRes[i].ID,
							"type":             postViewRes[i].Type,
							"period":           postViewRes[i].Period,
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

					record := core.NewRecord(schematicViewsCollection)
					record.Set("old_schematic_id", postViewRes[i].ID)
					record.Set("schematic", newSchematicID)
					record.Set("count", postViewRes[i].Count_)
					record.Set("type", postViewRes[i].Type)
					record.Set("period", postViewRes[i].Period)

					if err = app.Save(record); err != nil {
						return err
					}
					updated++
				}
			}
			return nil
		})
		if viewErr != nil {
			panic(viewErr)
		}
	}
	log.Printf("%d views migrated.\n", updated)
}
