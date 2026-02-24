package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}

		// Add featured boolean field
		collection.Fields.Add(&core.BoolField{
			Name: "featured",
		})

		// Add materials text field (stores JSON array)
		collection.Fields.Add(&core.TextField{
			Name: "materials",
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("featured")
		collection.Fields.RemoveByName("materials")

		return app.Save(collection)
	})
}
