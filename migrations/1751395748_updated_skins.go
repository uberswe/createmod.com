package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_709178859")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_3OdtDrX7bT` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `author` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_FRd7Pl5mjP` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_1v9LBZ1BZd` + "`" + ` ON ` + "`" + `skins` + "`" + ` (\n  ` + "`" + `title` + "`" + `,\n  ` + "`" + `description` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_PdMSmlZz7L` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `downloads` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_lpCiNlbJOS` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `views` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_QbkDJzOQB8` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `status` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_HEz8Kvi7PE` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `skin_type` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_RCeEkoyESX` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `skin_resolution` + "`" + `)",
				"CREATE UNIQUE INDEX ` + "`" + `idx_49RikXaC4O` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `slug` + "`" + `)"
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
		collection, err := app.FindCollectionByNameOrId("pbc_709178859")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_3OdtDrX7bT` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `author` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_FRd7Pl5mjP` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_1v9LBZ1BZd` + "`" + ` ON ` + "`" + `skins` + "`" + ` (\n  ` + "`" + `title` + "`" + `,\n  ` + "`" + `description` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_PdMSmlZz7L` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `downloads` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_lpCiNlbJOS` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `views` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_QbkDJzOQB8` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `status` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_HEz8Kvi7PE` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `skin_type` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_RCeEkoyESX` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `skin_resolution` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("text2560465762")

		return app.Save(collection)
	})
}
