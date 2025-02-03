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

		collection, err := dao.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX `+"`"+`idx_Q4u8xRs`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`title`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_XjMXYqQ`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`content`+"`"+`)",
			"CREATE INDEX `+"`"+`idx_Kl81wCh`+"`"+` ON `+"`"+`schematics`+"`"+` (`+"`"+`description`+"`"+`)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
