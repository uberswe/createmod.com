package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds a "published" bool field to the collections table.
// Default false — existing collections remain private until explicitly published.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("collections")
		if err != nil {
			return err
		}

		// add "published" bool field
		if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
			"hidden": false,
			"id": "bool_published",
			"name": "published",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("collections")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("bool_published")

		return app.Save(collection)
	})
}
