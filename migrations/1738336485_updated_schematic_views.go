package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
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
			"CREATE INDEX `+"`"+`idx_5KqiQGu`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`schematic`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_e44svrU`+"`"+` ON `+"`"+`schematic_views`+"`"+` (\n  `+"`"+`type`+"`"+`,\n  `+"`"+`period`+"`"+`,\n  `+"`"+`schematic`+"`"+`\n)",
			"CREATE INDEX `+"`"+`idx_WmSZio1`+"`"+` ON `+"`"+`schematic_views`+"`"+` (\n  `+"`"+`schematic`+"`"+`,\n  `+"`"+`type`+"`"+`\n)",
			"CREATE INDEX `+"`"+`idx_VKFrQb9`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`period`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_kYd1mBJ`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`type`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_KglTlZZ`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`schematic`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_ZOiPNYn`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`created`+"`"+`)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("2fqg0ilzru6uk5v")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX `+"`"+`idx_KmBXxNe`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`type`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_trHVTTI`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`period`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_5KqiQGu`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`schematic`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_e44svrU`+"`"+` ON `+"`"+`schematic_views`+"`"+` (\n  `+"`"+`type`+"`"+`,\n  `+"`"+`period`+"`"+`,\n  `+"`"+`schematic`+"`"+`\n)",
			"CREATE INDEX `+"`"+`idx_WmSZio1`+"`"+` ON `+"`"+`schematic_views`+"`"+` (\n  `+"`"+`schematic`+"`"+`,\n  `+"`"+`type`+"`"+`\n)",
			"CREATE INDEX `+"`"+`idx_VKFrQb9`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`period`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_kYd1mBJ`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`type`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_KglTlZZ`+"`"+` ON `+"`"+`schematic_views`+"`"+` (`+"`"+`schematic`+"`"+`)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
