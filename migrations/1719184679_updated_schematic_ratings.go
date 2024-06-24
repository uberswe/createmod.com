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
		new_old_schematic_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "jxaizwvt",
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

		// add
		new_old_user_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "fan72ogc",
			"name": "old_user_id",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_old_user_id); err != nil {
			return err
		}
		collection.Schema.AddField(new_old_user_id)

		// add
		new_old_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "kux2wdut",
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

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("dalbc2ck0f7127e")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("jxaizwvt")

		// remove
		collection.Schema.RemoveField("fan72ogc")

		// remove
		collection.Schema.RemoveField("kux2wdut")

		return dao.SaveCollection(collection)
	})
}
