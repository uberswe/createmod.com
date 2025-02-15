package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pm38mnqmbr4vlwz")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_hP5u33H` + "`" + ` ON ` + "`" + `schematic_tags` + "`" + ` (` + "`" + `key` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_KuaKNQYstU` + "`" + ` ON ` + "`" + `schematic_tags` + "`" + ` (` + "`" + `name` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pm38mnqmbr4vlwz")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_hP5u33H` + "`" + ` ON ` + "`" + `schematic_tags` + "`" + ` (` + "`" + `key` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
