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
		// Add datetime field scheduled_at (hidden)
		if err := collection.Fields.AddMarshaledJSONAt(32, []byte(`{
            "hidden": true,
            "id": "datetime_scheduled_at",
            "name": "scheduled_at",
            "onCreate": false,
            "onUpdate": false,
            "presentable": false,
            "required": false,
            "system": false,
            "type": "date"
        }`)); err != nil {
			return err
		}
		return app.Save(collection)
	}, func(app core.App) error {
		// Non-destructive down: keep field to avoid data loss
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}
		return app.Save(collection)
	})
}
