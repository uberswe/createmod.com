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

		collection, err := dao.FindCollectionByNameOrId("ovsujo1r5i1d4eg")
		if err != nil {
			return err
		}

		// add
		new_old_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "eftgi5bo",
			"name": "old_id",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_old_id); err != nil {
			return err
		}
		collection.Schema.AddField(new_old_id)

		// add
		new_old_schematic_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "j8dhhhql",
			"name": "old_schematic_id",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_old_schematic_id); err != nil {
			return err
		}
		collection.Schema.AddField(new_old_schematic_id)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("ovsujo1r5i1d4eg")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("eftgi5bo")

		// remove
		collection.Schema.RemoveField("j8dhhhql")

		return dao.SaveCollection(collection)
	})
}
