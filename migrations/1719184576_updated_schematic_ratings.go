package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("dalbc2ck0f7127e")
		if err != nil {
			return err
		}

		// add
		new_rated_at := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gvii5aas",
			"name": "rated_at",
			"type": "date",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": "",
				"max": ""
			}
		}`), new_rated_at); err != nil {
			return err
		}
		collection.Schema.AddField(new_rated_at)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("dalbc2ck0f7127e")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("gvii5aas")

		return dao.SaveCollection(collection)
	})
}
