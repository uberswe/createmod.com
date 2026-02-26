package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("schematic_files")
		if err != nil {
			return err
		}

		// Add text field: description
		descJSON := `{"autogeneratePattern":"","hidden":false,"id":"text_sf_description","max":0,"min":0,"name":"description","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"}`
		tf := &core.TextField{}
		if err := json.Unmarshal([]byte(descJSON), tf); err != nil {
			return err
		}
		collection.Fields.Add(tf)

		// Add file field: nbt_file
		fileJSON := `{"hidden":false,"id":"file_sf_nbt_file","maxSelect":1,"maxSize":10485760,"mimeTypes":[],"name":"nbt_file","presentable":false,"protected":false,"required":false,"system":false,"thumbs":[],"type":"file"}`
		ff := &core.FileField{}
		if err := json.Unmarshal([]byte(fileJSON), ff); err != nil {
			return err
		}
		collection.Fields.Add(ff)

		// Add number fields: block_count, dim_x, dim_y, dim_z
		numberFields := []string{
			`{"hidden":false,"id":"number_sf_block_count","max":null,"min":null,"name":"block_count","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_sf_dim_x","max":null,"min":null,"name":"dim_x","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_sf_dim_y","max":null,"min":null,"name":"dim_y","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_sf_dim_z","max":null,"min":null,"name":"dim_z","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
		}
		for _, fj := range numberFields {
			f := &core.NumberField{}
			if err := json.Unmarshal([]byte(fj), f); err != nil {
				return err
			}
			collection.Fields.Add(f)
		}

		// Add JSON field: materials
		matJSON := `{"hidden":false,"id":"json_sf_materials","maxSize":0,"name":"materials","presentable":false,"required":false,"system":false,"type":"json"}`
		jf := &core.JSONField{}
		if err := json.Unmarshal([]byte(matJSON), jf); err != nil {
			return err
		}
		collection.Fields.Add(jf)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("schematic_files")
		if err != nil {
			return err
		}
		ids := []string{
			"text_sf_description",
			"file_sf_nbt_file",
			"number_sf_block_count",
			"number_sf_dim_x",
			"number_sf_dim_y",
			"number_sf_dim_z",
			"json_sf_materials",
		}
		for _, id := range ids {
			collection.Fields.RemoveById(id)
		}
		return app.Save(collection)
	})
}
