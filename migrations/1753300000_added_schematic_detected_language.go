package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds a detected_language text field to schematics
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}
		// add field (hidden=false, simple text)
		if err := collection.Fields.AddMarshaledJSONAt(61, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text_detected_language",
			"max": 0,
			"min": 0,
			"name": "detected_language",
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
		collection.Fields.RemoveById("text_detected_language")
		return app.Save(collection)
	})
}
