package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `{
			"createRule": null,
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text_id_pk",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text_checksum",
					"max": 64,
					"min": 64,
					"name": "checksum",
					"pattern": "^[a-f0-9]+$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text_schematic",
					"max": 0,
					"min": 0,
					"name": "schematic",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text_uploaded_by",
					"max": 0,
					"min": 0,
					"name": "uploaded_by",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "autodate_created",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_nbt_hashes",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + "idx_nbt_hashes_checksum" + "`" + ` ON ` + "`" + "nbt_hashes" + "`" + ` (` + "`" + "checksum" + "`" + `)",
				"CREATE INDEX ` + "`" + "idx_nbt_hashes_uploaded_by" + "`" + ` ON ` + "`" + "nbt_hashes" + "`" + ` (` + "`" + "uploaded_by" + "`" + `)",
				"CREATE INDEX ` + "`" + "idx_nbt_hashes_created" + "`" + ` ON ` + "`" + "nbt_hashes" + "`" + ` (` + "`" + "created" + "`" + `)"
			],
			"listRule": null,
			"name": "nbt_hashes",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("nbt_hashes")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
