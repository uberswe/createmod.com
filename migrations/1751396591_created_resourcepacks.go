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
					"id": "file1704208859",
					"maxSelect": 1,
					"maxSize": 0,
					"mimeTypes": [],
					"name": "icon",
					"presentable": false,
					"protected": false,
					"required": false,
					"system": false,
					"thumbs": [],
					"type": "file"
				},
				{
					"hidden": false,
					"id": "file3160978512",
					"maxSelect": 1,
					"maxSize": 0,
					"mimeTypes": [],
					"name": "background",
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
					"cascadeDelete": false,
					"collectionId": "qj5v20zrmwxtnff",
					"hidden": false,
					"id": "relation107458750",
					"maxSelect": 999,
					"minSelect": 0,
					"name": "minecraft_versions",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"exceptDomains": null,
					"hidden": false,
					"id": "url2677815203",
					"name": "curseforge_url",
					"onlyDomains": null,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "url"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text1041931776",
					"max": 0,
					"min": 0,
					"name": "curseforge_id",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"exceptDomains": null,
					"hidden": false,
					"id": "url3185113621",
					"name": "modrinth_url",
					"onlyDomains": null,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "url"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text2025951670",
					"max": 0,
					"min": 0,
					"name": "modrinth_id",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
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
			"id": "pbc_47156727",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_unCHSIwyuV` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `slug` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_FckNCSHo41` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_kxLJTGHJyz` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (\n  ` + "`" + `description` + "`" + `,\n  ` + "`" + `title` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_l8uZCyKMrg` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `minecraft_versions` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_9CdwzmVZMj` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `curseforge_id` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_aICoJFUSYw` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `modrinth_id` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_VGk0ZOWNQp` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `status` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_UqP4A0h5UP` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `views` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_jVt8P7GTVY` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `downloads` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_mTvrXYaovm` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `created` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_gyhvRbk80d` + "`" + ` ON ` + "`" + `resourcepacks` + "`" + ` (` + "`" + `updated` + "`" + `)"
			],
			"listRule": null,
			"name": "resourcepacks",
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
		collection, err := app.FindCollectionByNameOrId("pbc_47156727")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
