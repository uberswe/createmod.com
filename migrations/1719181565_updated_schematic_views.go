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

		collection, err := dao.FindCollectionByNameOrId("2fqg0ilzru6uk5v")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX `+"`"+`idx_KmBXxNe`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`type`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_trHVTTI`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`period`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_5KqiQGu`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`schematic`+"`"+`)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// add
		new_old_schematic_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "5ppmhxzs",
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
		new_schematic := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "x43yvmir",
			"name": "schematic",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "ezzomjw4q1qibza",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_schematic); err != nil {
			return err
		}
		collection.Schema.AddField(new_schematic)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("2fqg0ilzru6uk5v")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX `+"`"+`idx_KmBXxNe`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`type`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_trHVTTI`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`period`+"`"+`)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("5ppmhxzs")

		// remove
		collection.Schema.RemoveField("x43yvmir")

		return dao.SaveCollection(collection)
	})
}
