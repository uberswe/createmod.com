package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// achievements collection
		achJSON := `{
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
					"id": "text_key",
					"max": 0,
					"min": 0,
					"name": "key",
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
					"id": "text_title",
					"max": 0,
					"min": 0,
					"name": "title",
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
					"id": "text_desc",
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
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text_icon",
					"max": 0,
					"min": 0,
					"name": "icon",
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
			"id": "pbc_achievements",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + "idx_achievements_key" + "`" + ` ON ` + "`" + "achievements" + "`" + ` (` + "`" + "key" + "`" + `)"
			],
			"listRule": null,
			"name": "achievements",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		}`

		ach := &core.Collection{}
		if err := json.Unmarshal([]byte(achJSON), &ach); err != nil {
			return err
		}
		if err := app.Save(ach); err != nil {
			return err
		}

		// user_achievements collection
		uaJSON := `{
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
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "rel_user",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "user",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": false,
					"collectionId": "pbc_achievements",
					"hidden": false,
					"id": "rel_ach",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "achievement",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
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
			"id": "pbc_user_achievements",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + "idx_user_achievements_user_achievement" + "`" + ` ON ` + "`" + "user_achievements" + "`" + ` (` + "`" + "user" + "`" + `, ` + "`" + "achievement" + "`" + `)"
			],
			"listRule": null,
			"name": "user_achievements",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		}`

		ua := &core.Collection{}
		if err := json.Unmarshal([]byte(uaJSON), &ua); err != nil {
			return err
		}
		return app.Save(ua)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("user_achievements"); err == nil {
			_ = app.Delete(c)
		}
		if c, err := app.FindCollectionByNameOrId("achievements"); err == nil {
			_ = app.Delete(c)
		}
		return nil
	})
}
