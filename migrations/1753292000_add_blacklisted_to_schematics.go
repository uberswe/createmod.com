package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// add bool field "blacklisted" (hidden)
		if err := collection.Fields.AddMarshaledJSONAt(31, []byte(`{
			"hidden": true,
			"id": "bool_blacklisted",
			"name": "blacklisted",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// best-effort down migration: leave field as-is to avoid accidental data loss
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}
		return app.Save(collection)
	})
}
