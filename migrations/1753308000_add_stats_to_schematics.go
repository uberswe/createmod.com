package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}

		// Add number fields: block_count, dim_x, dim_y, dim_z
		numberFields := []string{
			`{"hidden":false,"id":"number_sch_block_count","max":null,"min":null,"name":"block_count","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_sch_dim_x","max":null,"min":null,"name":"dim_x","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_sch_dim_y","max":null,"min":null,"name":"dim_y","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_sch_dim_z","max":null,"min":null,"name":"dim_z","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
		}
		for _, fj := range numberFields {
			f := &core.NumberField{}
			if err := json.Unmarshal([]byte(fj), f); err != nil {
				return err
			}
			collection.Fields.Add(f)
		}

		// Add JSON field: mods
		modsJSON := `{"hidden":false,"id":"json_sch_mods","maxSize":0,"name":"mods","presentable":false,"required":false,"system":false,"type":"json"}`
		jf := &core.JSONField{}
		if err := json.Unmarshal([]byte(modsJSON), jf); err != nil {
			return err
		}
		collection.Fields.Add(jf)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		ids := []string{
			"number_sch_block_count",
			"number_sch_dim_x",
			"number_sch_dim_y",
			"number_sch_dim_z",
			"json_sch_mods",
		}
		for _, id := range ids {
			collection.Fields.RemoveById(id)
		}
		return app.Save(collection)
	})
}
