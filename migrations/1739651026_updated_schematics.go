package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_Q4u8xRs` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_XjMXYqQ` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `content` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_Kl81wCh` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `description` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_nVVYTrB2iQ` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `type` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_N9h84HE0qA` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `deleted` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_oZwOAMtvJs` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `created` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_Q4u8xRs` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_XjMXYqQ` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `content` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_Kl81wCh` + "`" + ` ON ` + "`" + `schematics` + "`" + ` (` + "`" + `description` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
