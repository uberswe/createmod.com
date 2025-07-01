package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3287471486")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_Q7ZO0BMH9M` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_UBt23573ib` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (\n  ` + "`" + `description` + "`" + `,\n  ` + "`" + `title` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_o4gdDaFX6s` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `status` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_JS6ecTePD4` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `gamemode` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_Phxu5CHvU4` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `created` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_4F45fSSLZh` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `updated` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_WZibxF2Fv5` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `author` + "`" + `)",
				"CREATE UNIQUE INDEX ` + "`" + `idx_ZUbIfdk5l1` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `slug` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text2560465762",
			"max": 0,
			"min": 0,
			"name": "slug",
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
		collection, err := app.FindCollectionByNameOrId("pbc_3287471486")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_Q7ZO0BMH9M` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_UBt23573ib` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (\n  ` + "`" + `description` + "`" + `,\n  ` + "`" + `title` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_o4gdDaFX6s` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `status` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_JS6ecTePD4` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `gamemode` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_Phxu5CHvU4` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `created` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_4F45fSSLZh` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `updated` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_WZibxF2Fv5` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `author` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("text2560465762")

		return app.Save(collection)
	})
}
