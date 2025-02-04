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
		edit_gallery := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ic3febng",
			"name": "gallery",
			"type": "file",
			"required": false,
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
					"640x480",
					"1280x720",
					"1920x1080"
				],
				"maxSelect": 99,
				"maxSize": 10485760,
				"protected": false
			}
		}`), edit_gallery); err != nil {
			return err
		}
		collection.Schema.AddField(edit_gallery)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update
		edit_gallery := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ic3febng",
			"name": "gallery",
			"type": "file",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"mimeTypes": [
					"image/png",
					"image/vnd.mozilla.apng",
					"image/jpeg",
					"image/webp"
				],
				"thumbs": [],
				"maxSelect": 99,
				"maxSize": 10485760,
				"protected": false
			}
		}`), edit_gallery); err != nil {
			return err
		}
		collection.Schema.AddField(edit_gallery)

		return dao.SaveCollection(collection)
	})
}
