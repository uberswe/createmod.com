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
					"id": "text_schematic",
					"max": 0,
					"min": 0,
					"name": "schematic",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "number_count",
					"max": null,
					"min": null,
					"name": "count",
					"onlyInt": true,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number_type",
					"max": null,
					"min": null,
					"name": "type",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text_period",
					"max": 0,
					"min": 0,
					"name": "period",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
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
				},
				{
					"hidden": false,
					"id": "autodate_updated",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_schematic_downloads",
			"indexes": [
				"CREATE INDEX ` + "`" + "idx_sdl_type_period_schematic" + "`" + ` ON ` + "`" + "schematic_downloads" + "`" + ` (\n  ` + "`" + "type" + "`" + `,\n  ` + "`" + "period" + "`" + `,\n  ` + "`" + "schematic" + "`" + `\n)",
				"CREATE INDEX ` + "`" + "idx_sdl_schematic_type" + "`" + ` ON ` + "`" + "schematic_downloads" + "`" + ` (\n  ` + "`" + "schematic" + "`" + `,\n  ` + "`" + "type" + "`" + `\n)",
				"CREATE INDEX ` + "`" + "idx_sdl_period" + "`" + ` ON ` + "`" + "schematic_downloads" + "`" + ` (` + "`" + "period" + "`" + `)",
				"CREATE INDEX ` + "`" + "idx_sdl_type" + "`" + ` ON ` + "`" + "schematic_downloads" + "`" + ` (` + "`" + "type" + "`" + `)",
				"CREATE INDEX ` + "`" + "idx_sdl_schematic" + "`" + ` ON ` + "`" + "schematic_downloads" + "`" + ` (` + "`" + "schematic" + "`" + `)",
				"CREATE INDEX ` + "`" + "idx_sdl_created" + "`" + ` ON ` + "`" + "schematic_downloads" + "`" + ` (` + "`" + "created" + "`" + `)"
			],
			"listRule": null,
			"name": "schematic_downloads",
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
		collection, err := app.FindCollectionByNameOrId("schematic_downloads")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
