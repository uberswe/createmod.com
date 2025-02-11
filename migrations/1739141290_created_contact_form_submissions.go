package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `{
			"id": "hrall4xkkfwg09d",
			"created": "2025-02-09 22:48:10.454Z",
			"updated": "2025-02-09 22:48:10.454Z",
			"name": "contact_form_submissions",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "0kcyc0vb",
					"name": "email",
					"type": "email",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"exceptDomains": null,
						"onlyDomains": null
					}
				},
				{
					"system": false,
					"id": "el7m9mtp",
					"name": "content",
					"type": "editor",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"convertUrls": false
					}
				}
			],
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_tLBOEl4` + "`" + ` ON ` + "`" + `contact_form_submissions` + "`" + ` (` + "`" + `email` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_E5ujN5K` + "`" + ` ON ` + "`" + `contact_form_submissions` + "`" + ` (` + "`" + `created` + "`" + `)"
			],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("hrall4xkkfwg09d")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
