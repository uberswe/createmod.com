package migrate

import (
	"github.com/pocketbase/pocketbase"
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID              string
	OldID           int64
	Created         time.Time
	Updated         time.Time
	Username        string
	Email           string
	OldPassword     string
	EmailVisibility bool
	Verified        bool
	Name            string
	Avatar          string
	URL             string
	Status          string
}

// Run migrates the mysql Wordpress database to pb sqlite
func Run(app *pocketbase.PocketBase, gormdb *gorm.DB) {
	app.Logger().Info("Running migration from Wordpress")
	userOldId := migrateUsers(app, gormdb)
	schematicOldId := migrateSchematics(app, gormdb, userOldId)
	migrateRatings(app, gormdb, userOldId, schematicOldId)
	migrateViews(app, gormdb, userOldId, schematicOldId)
	migrateComments(app, gormdb, userOldId, schematicOldId)
	app.Logger().Info("Migrations from Wordpress complete")
}
