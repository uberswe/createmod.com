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
		if err := collection.Fields.AddMarshaledJSONAt(38, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text25424793",
			"max": 0,
			"min": 0,
			"name": "ai_description",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
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
		collection.Fields.RemoveById("text25424793")

		return app.Save(collection)
	})
}
