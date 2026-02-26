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
					"id": "text_url",
					"max": 0,
					"min": 0,
					"name": "url",
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
					"id": "text_source_type",
					"max": 0,
					"min": 0,
					"name": "source_type",
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
					"id": "text_source_id",
					"max": 0,
					"min": 0,
					"name": "source_id",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "num_clicks",
					"max": null,
					"min": 0,
					"name": "clicks",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
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
			"id": "pbc_outgoing_clicks",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_outgoing_clicks_unique` + "`" + ` ON ` + "`" + `outgoing_clicks` + "`" + ` (` + "`" + `url` + "`" + `, ` + "`" + `source_type` + "`" + `, ` + "`" + `source_id` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_outgoing_clicks_source_type` + "`" + ` ON ` + "`" + `outgoing_clicks` + "`" + ` (` + "`" + `source_type` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_outgoing_clicks_source_id` + "`" + ` ON ` + "`" + `outgoing_clicks` + "`" + ` (` + "`" + `source_id` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_outgoing_clicks_clicks` + "`" + ` ON ` + "`" + `outgoing_clicks` + "`" + ` (` + "`" + `clicks` + "`" + `)"
			],
			"listRule": null,
			"name": "outgoing_clicks",
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
		collection, err := app.FindCollectionByNameOrId("outgoing_clicks")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
