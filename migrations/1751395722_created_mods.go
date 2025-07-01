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
					"required": true,
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
					"exceptDomains": null,
					"hidden": false,
					"id": "url1215696702",
					"name": "curseforge_link",
					"onlyDomains": null,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "url"
				},
				{
					"exceptDomains": null,
					"hidden": false,
					"id": "url2144677644",
					"name": "curseforge_link_alt",
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
					"id": "url1784044874",
					"name": "modrinth_link",
					"onlyDomains": null,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "url"
				},
				{
					"exceptDomains": null,
					"hidden": false,
					"id": "url3025438559",
					"name": "modrinth_link_alt",
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
					"id": "select1645843298",
					"maxSelect": 1,
					"name": "mod_loaders",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"forge",
						"neoforge",
						"fabric"
					]
				},
				{
					"exceptDomains": null,
					"hidden": false,
					"id": "url1198480871",
					"name": "website",
					"onlyDomains": null,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "url"
				},
				{
					"hidden": false,
					"id": "json4065306594",
					"maxSize": 0,
					"name": "wikis",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
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
			"id": "pbc_1142134090",
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_ahD2eSMDC5` + "`" + ` ON ` + "`" + `mods` + "`" + ` (` + "`" + `title` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_dQGCQdSe3Q` + "`" + ` ON ` + "`" + `mods` + "`" + ` (\n  ` + "`" + `description` + "`" + `,\n  ` + "`" + `title` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_RRzpijntFY` + "`" + ` ON ` + "`" + `mods` + "`" + ` (` + "`" + `author` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_S1iGKKTLOQ` + "`" + ` ON ` + "`" + `mods` + "`" + ` (` + "`" + `curseforge_id` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_7Hs3eQ9Lx1` + "`" + ` ON ` + "`" + `mods` + "`" + ` (` + "`" + `modrinth_id` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_4dKABjKaeE` + "`" + ` ON ` + "`" + `mods` + "`" + ` (` + "`" + `mod_loaders` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_AhEewKxcmo` + "`" + ` ON ` + "`" + `mods` + "`" + ` (` + "`" + `created` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_rizFEFoJzw` + "`" + ` ON ` + "`" + `mods` + "`" + ` (` + "`" + `updated` + "`" + `)",
				"CREATE UNIQUE INDEX ` + "`" + `idx_MLNgt1Eddz` + "`" + ` ON ` + "`" + `mods` + "`" + ` (` + "`" + `slug` + "`" + `)"
			],
			"listRule": null,
			"name": "mods",
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
		collection, err := app.FindCollectionByNameOrId("pbc_1142134090")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
