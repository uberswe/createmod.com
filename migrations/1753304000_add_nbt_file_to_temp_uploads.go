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

		// Add nbt_file field
		fieldJSON := `{
			"hidden": false,
			"id": "file_nbt",
			"maxSelect": 1,
			"maxSize": 10485760,
			"mimeTypes": [],
			"name": "nbt_file",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": [],
			"type": "file"
		}`

		newField := &core.FileField{}
		if err := json.Unmarshal([]byte(fieldJSON), newField); err != nil {
			return err
		}
		collection.Fields.Add(newField)

		// Allow public read access so Bloxelizer can fetch the NBT file
		emptyRule := ""
		collection.ViewRule = &emptyRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("temp_uploads")
		if err != nil {
			return err
		}
		collection.Fields.RemoveById("file_nbt")
		return app.Save(collection)
	})
}
