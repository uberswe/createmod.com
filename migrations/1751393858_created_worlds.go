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
					"id": "file1194031162",
					"maxSelect": 99,
					"maxSize": 0,
					"mimeTypes": [],
					"name": "gallery",
					"presentable": false,
					"protected": false,
					"required": false,
					"system": false,
					"thumbs": [],
					"type": "file"
				},
				{
					"hidden": false,
					"id": "file1381758148",
					"maxSelect": 1,
					"maxSize": 0,
					"mimeTypes": [],
					"name": "world_file",
					"presentable": false,
					"protected": false,
					"required": false,
					"system": false,
					"thumbs": [],
					"type": "file"
				},
				{
					"cascadeDelete": false,
					"collectionId": "qj5v20zrmwxtnff",
					"hidden": false,
					"id": "relation3731869968",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "minecraft_version",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
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
						"pending",
						"denied"
					]
				},
				{
					"hidden": false,
					"id": "select3254472084",
					"maxSelect": 1,
					"name": "gamemode",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"creative",
						"peaceful",
						"easy",
						"normal",
						"hard",
						"hardcore"
					]
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text1149756166",
					"max": 0,
					"min": 0,
					"name": "seed",
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
					"id": "text688187070",
					"max": 0,
					"min": 0,
					"name": "world_size",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
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
			"id": "pbc_3287471486",
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_Q7ZO0BMH9M` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_UBt23573ib` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (\n  ` + "`" + `description` + "`" + `,\n  ` + "`" + `title` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_o4gdDaFX6s` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `status` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_JS6ecTePD4` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `gamemode` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_Phxu5CHvU4` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `created` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_4F45fSSLZh` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `updated` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_WZibxF2Fv5` + "`" + ` ON ` + "`" + `worlds` + "`" + ` (` + "`" + `author` + "`" + `)"
			],
			"listRule": null,
			"name": "worlds",
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
		collection, err := app.FindCollectionByNameOrId("pbc_3287471486")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
