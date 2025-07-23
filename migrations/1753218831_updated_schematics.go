package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX `+"`"+`idx_Q4u8xRs`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`title`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_XjMXYqQ`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`content`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_Kl81wCh`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`description`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_nVVYTrB2iQ`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`type`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_N9h84HE0qA`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`deleted`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_oZwOAMtvJs`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`created`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_rjGyyyAzVq`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`moderated`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(36, []byte(`{
			"hidden": false,
			"id": "bool1678503859",
			"name": "moderated",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(37, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text3425273270",
			"max": 0,
			"min": 0,
			"name": "moderation_reason",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		err = app.Save(collection)
		if err != nil {
			return err
		}

		// Loop schematics and set moderated to true
		schematics, err := app.FindRecordsByFilter("schematics", "deleted = null", "-created", -1, 0)
		if err != nil {
			return err
		}
		for _, s := range schematics {
			s.Set("moderated", 1)
			err = app.Save(s)
			if err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX `+"`"+`idx_Q4u8xRs`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`title`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_XjMXYqQ`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`content`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_Kl81wCh`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`description`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_nVVYTrB2iQ`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`type`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_N9h84HE0qA`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`deleted`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_oZwOAMtvJs`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`created`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("bool1678503859")

		// remove field
		collection.Fields.RemoveById("text3425273270")

		return app.Save(collection)
	})
}
