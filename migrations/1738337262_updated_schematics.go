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

		collection, err := dao.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update
		edit_featured_image := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ptuuoygx",
			"name": "featured_image",
			"type": "file",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"mimeTypes": [
					"image/png",
					"image/vnd.mozilla.apng",
					"image/jpeg",
					"image/webp"
				],
				"thumbs": [
					"150x150",
					"600x600",
					"1000x1000",
					"1280x720",
					"1920x1080",
					"640x480"
				],
				"maxSelect": 1,
				"maxSize": 10485760,
				"protected": false
			}
		}`), edit_featured_image); err != nil {
			return err
		}
		collection.Schema.AddField(edit_featured_image)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update
		edit_featured_image := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ptuuoygx",
			"name": "featured_image",
			"type": "file",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"mimeTypes": [
					"image/png",
					"image/vnd.mozilla.apng",
					"image/jpeg",
					"image/webp"
				],
				"thumbs": [
					"150x150",
					"600x600",
					"1000x1000",
					"1280x720",
					"1920x1080"
				],
				"maxSelect": 1,
				"maxSize": 10485760,
				"protected": false
			}
		}`), edit_featured_image); err != nil {
			return err
		}
		collection.Schema.AddField(edit_featured_image)

		return dao.SaveCollection(collection)
	})
}
