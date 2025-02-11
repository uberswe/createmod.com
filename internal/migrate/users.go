package migrate

import (
	"createmod/query"
	"fmt"
	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"gorm.io/gorm"
	"log"
)

func migrateUsers(app *pocketbase.PocketBase, gormdb *gorm.DB) (userOldId map[int64]string) {
	q := query.Use(gormdb)
	res, err := q.QeyKryWEuser.Find()
	if err != nil {
		panic(err)
	}
	userCollection, err := app.Dao().FindCollectionByNameOrId("users")
	if err != nil {
		panic(err)
	}
	log.Println("Migrating users.")
	userOldId = make(map[int64]string, len(res))

	for _, u := range res {
		var user User
		userErr := app.Dao().DB().
			NewQuery("SELECT id, old_id FROM users WHERE old_id={:old_id}").
			Bind(dbx.Params{
				"old_id": u.ID,
			}).
			One(&user)
		if userErr == nil {
			app.Logger().Debug(
				"Skipping user migration, record exists",
				"user-id", user.OldID,
			)
			userOldId[u.ID] = user.ID
			continue
		}

		record := models.NewRecord(userCollection)

		record.Set("old_id", u.ID)
		record.Set("created", u.UserRegistered)
		record.Set("username", u.UserNicename)
		record.Set("email", u.UserEmail)
		record.Set("old_password", u.UserPass) // We can't and don't want to know the old password, will force reset all users later and remove this
		record.Set("name", u.DisplayName)
		record.Set("url", u.UserURL)
		record.Set("status", fmt.Sprintf("%d", u.UserStatus))
		record.Set("tokenKey", uuid.Must(uuid.NewRandom()).String())

		if err := app.Dao().SaveRecord(record); err != nil {
			panic(err)
		}
		userOldId[u.ID] = record.GetId()
	}
	log.Printf("%d users processed.\n", len(userOldId))
	return userOldId
}
