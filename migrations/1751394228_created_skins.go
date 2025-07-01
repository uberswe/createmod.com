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
					"id": "text3208210256",
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
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "relation3182418120",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "author",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text724990059",
					"max": 0,
					"min": 0,
					"name": "title",
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
					"id": "text1843675174",
					"max": 0,
					"min": 0,
					"name": "description",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "file2624913349",
					"maxSelect": 1,
					"maxSize": 0,
					"mimeTypes": [],
					"name": "featured_image",
					"presentable": false,
					"protected": false,
					"required": false,
					"system": false,
					"thumbs": [],
					"type": "file"
				},
				{
					"hidden": false,
					"id": "file2582544729",
					"maxSelect": 1,
					"maxSize": 0,
					"mimeTypes": [],
					"name": "skin_file",
					"presentable": false,
					"protected": false,
					"required": false,
					"system": false,
					"thumbs": [],
					"type": "file"
				},
				{
					"hidden": false,
					"id": "number1265870005",
					"max": null,
					"min": null,
					"name": "downloads",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number300981383",
					"max": null,
					"min": null,
					"name": "views",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "select2063623452",
					"maxSelect": 1,
					"name": "status",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"approved",
						"denied",
						"pending"
					]
				},
				{
					"hidden": false,
					"id": "select2578440288",
					"maxSelect": 1,
					"name": "skin_type",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"classic",
						"slim"
					]
				},
				{
					"hidden": false,
					"id": "select3188532808",
					"maxSelect": 1,
					"name": "skin_resolution",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"16x16",
						"32x32",
						"64x64",
						"128x128",
						"HD"
					]
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_709178859",
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_3OdtDrX7bT` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `author` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_FRd7Pl5mjP` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_1v9LBZ1BZd` + "`" + ` ON ` + "`" + `skins` + "`" + ` (\n  ` + "`" + `title` + "`" + `,\n  ` + "`" + `description` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_PdMSmlZz7L` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `downloads` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_lpCiNlbJOS` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `views` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_QbkDJzOQB8` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `status` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_HEz8Kvi7PE` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `skin_type` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_RCeEkoyESX` + "`" + ` ON ` + "`" + `skins` + "`" + ` (` + "`" + `skin_resolution` + "`" + `)"
			],
			"listRule": null,
			"name": "skins",
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
		collection, err := app.FindCollectionByNameOrId("pbc_709178859")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
