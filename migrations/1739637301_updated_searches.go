package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_542531584")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX `+"`"+`idx_medmtWj1Iz`+"`"+` ON `+"`"+`searches`+"`"+` (`+"`"+`term`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_97uBOIspKc`+"`"+` ON `+"`"+`searches`+"`"+` (`+"`"+`slug`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_5z8m9oGCO6`+"`"+` ON `+"`"+`searches`+"`"+` (`+"`"+`searches`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_WvzE8RTAdj`+"`"+` ON `+"`"+`searches`+"`"+` (`+"`"+`results`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_542531584")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX `+"`"+`idx_medmtWj1Iz`+"`"+` ON `+"`"+`searches`+"`"+` (`+"`"+`term`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_97uBOIspKc`+"`"+` ON `+"`"+`searches`+"`"+` (`+"`"+`slug`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
