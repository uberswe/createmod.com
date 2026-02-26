package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("temp_uploads")
		if err != nil {
			return err
		}

		// Add number fields: block_count, dim_x, dim_y, dim_z
		numberFields := []string{
			`{"hidden":false,"id":"number_block_count","max":null,"min":null,"name":"block_count","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_dim_x","max":null,"min":null,"name":"dim_x","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_dim_y","max":null,"min":null,"name":"dim_y","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
			`{"hidden":false,"id":"number_dim_z","max":null,"min":null,"name":"dim_z","onlyInt":true,"presentable":false,"required":false,"system":false,"type":"number"}`,
		}
		for _, fj := range numberFields {
			f := &core.NumberField{}
			if err := json.Unmarshal([]byte(fj), f); err != nil {
				return err
			}
			collection.Fields.Add(f)
		}

		// Add JSON fields: mods, materials
		jsonFields := []string{
			`{"hidden":false,"id":"json_mods","maxSize":0,"name":"mods","presentable":false,"required":false,"system":false,"type":"json"}`,
			`{"hidden":false,"id":"json_materials","maxSize":0,"name":"materials","presentable":false,"required":false,"system":false,"type":"json"}`,
		}
		for _, fj := range jsonFields {
			f := &core.JSONField{}
			if err := json.Unmarshal([]byte(fj), f); err != nil {
				return err
			}
			collection.Fields.Add(f)
		}

		// Add text field: uploaded_by
		uploadedByJSON := `{"autogeneratePattern":"","hidden":false,"id":"text_uploaded_by","max":0,"min":0,"name":"uploaded_by","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"}`
		tf := &core.TextField{}
		if err := json.Unmarshal([]byte(uploadedByJSON), tf); err != nil {
			return err
		}
		collection.Fields.Add(tf)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("temp_uploads")
		if err != nil {
			return err
		}
		ids := []string{
			"number_block_count",
			"number_dim_x",
			"number_dim_y",
			"number_dim_z",
			"json_mods",
			"json_materials",
			"text_uploaded_by",
		}
		for _, id := range ids {
			collection.Fields.RemoveById(id)
		}
		return app.Save(collection)
	})
}
