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

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(35, []byte(`{
			"hidden": true,
			"id": "date3946532403",
			"max": "",
			"min": "",
			"name": "deleted",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("date3946532403")

		return app.Save(collection)
	})
}
