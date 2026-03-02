package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("api_key_usage")
		if err != nil {
			return err
		}

		// Add "endpoint" text field for per-endpoint tracking
		collection.Fields.Add(&core.TextField{
			Id:       "text_endpoint",
			Name:     "endpoint",
			Required: false,
		})

		// Update index to include endpoint for composite lookups
		collection.RemoveIndex("idx_api_key_usage_key")
		collection.AddIndex("idx_api_key_usage_key_endpoint", false, "key, endpoint", "")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("api_key_usage")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("text_endpoint")
		collection.RemoveIndex("idx_api_key_usage_key_endpoint")
		collection.AddIndex("idx_api_key_usage_key", false, "key", "")

		return app.Save(collection)
	})
}
