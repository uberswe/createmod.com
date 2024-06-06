package migrate

import (
	"createmod/query"
	"github.com/pocketbase/pocketbase"
	"gorm.io/gorm"
	"log"
)

// Run migrates the mysql database to pb sqlite
func Run(app *pocketbase.PocketBase, gormdb *gorm.DB) {
	q := query.Use(gormdb)
	users, err := q.QeyKryWEuser.Find()
	if err != nil {
		panic(err)
	}
	for _, u := range users {
		log.Println(u.UserEmail)
	}
}
