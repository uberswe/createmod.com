package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("frpelz2jx9v0q7b")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_TbxtwFC` + "`" + ` ON ` + "`" + `schematic_categories` + "`" + ` (` + "`" + `key` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_gZFcmodoLk` + "`" + ` ON ` + "`" + `schematic_categories` + "`" + ` (` + "`" + `name` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("frpelz2jx9v0q7b")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_TbxtwFC` + "`" + ` ON ` + "`" + `schematic_categories` + "`" + ` (` + "`" + `key` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
