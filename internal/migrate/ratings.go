package migrate

import (
	"createmod/query"
	"errors"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"gorm.io/gorm"
	"log"
	"time"
)

type viewMigration struct {
	OldID     int64
	OldUserId int64
	OldPostId int64
	Date      time.Time
	Value     int32
}

func migrateRatings(app *pocketbase.PocketBase, gormdb *gorm.DB, oldUserIDs map[int64]string, oldSchematicIDs map[int64]string) {
	log.Println("Migrating ratings.")

	// QeyKryWEmr_rating_item_entry
	// user_id
	// post_id
	// date

	// QeyKryWEmr_rating_item_entry_value
	// rating_item_entry_id
	// value

	q := query.Use(gormdb)
	ratingEntries, ratingErr := q.QeyKryWEmrRatingItemEntry.Find()
	if ratingErr != nil {
		panic(ratingErr)
	}

	valuesEntries, valuesErr := q.QeyKryWEmrRatingItemEntryValue.Find()
	if valuesErr != nil {
		panic(valuesErr)
	}

	migrations := make([]viewMigration, 0)

	schematicRatingsCollection, err := app.FindCollectionByNameOrId("schematic_ratings")
	if err != nil {
		panic(err)
	}

	totalCount := res{}
	countErr := app.DB().
		NewQuery("SELECT COUNT(id) as c FROM schematic_ratings").
		One(&totalCount)
	if countErr != nil {
		panic(countErr)
	}

	log.Println("Total ratings count:", totalCount.C)
	if totalCount.C >= int64(len(ratingEntries)) {
		log.Println("Skipping ratings, already migrated.")
		return
	}

	for _, e := range ratingEntries {
		for _, v := range valuesEntries {
			if e.RatingItemEntryID == v.RatingItemEntryID {
				migrations = append(migrations, viewMigration{
					OldID:     e.RatingItemEntryID,
					OldUserId: e.UserID,
					OldPostId: e.PostID,
					Date:      e.EntryDate,
					Value:     v.Value,
				})
			}
		}
	}
	updated := 0
	for _, vm := range migrations {
		filter, err := app.FindRecordsByFilter(
			schematicRatingsCollection.Id,
			"old_id = {:old_id}",
			"-created",
			1,
			0,
			dbx.Params{"old_id": vm.OldID})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			app.Logger().Debug(
				fmt.Sprintf("Rating error: %v", err),
				"filter-len", len(filter),
			)
			continue
		} else if err == nil && len(filter) != 0 {
			app.Logger().Debug(
				fmt.Sprintf("Rating found: %v", err),
				"filter-len", len(filter),
			)
			return
		}

		newUserId := oldUserIDs[vm.OldUserId]
		newSchematicId := oldSchematicIDs[vm.OldPostId]
		record := core.NewRecord(schematicRatingsCollection)
		record.Set("rated_at", vm.Date)
		record.Set("old_id", vm.OldID)
		record.Set("old_schematic_id", vm.OldPostId)
		record.Set("old_user_id", vm.OldUserId)
		record.Set("rating", vm.Value)
		record.Set("user", newUserId)
		record.Set("schematic", newSchematicId)
		if err = app.Save(record); err != nil {
			log.Printf("ERROR for %d - %d: %v\n", vm.Value, vm.OldID, err)
			continue
		}
		updated++
	}
	log.Printf("%d ratings migrated.\n", updated)
}
