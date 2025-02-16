package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(25, []byte(`{
			"hidden": false,
			"id": "ptuuoygx",
			"maxSelect": 1,
			"maxSize": 26214400,
			"mimeTypes": [
				"image/png",
				"image/vnd.mozilla.apng",
				"image/jpeg",
				"image/webp"
			],
			"name": "featured_image",
			"presentable": false,
			"protected": false,
			"required": true,
			"system": false,
			"thumbs": [
				"150x150",
				"1920x1080",
				"640x360",
				"320x180"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(26, []byte(`{
			"hidden": false,
			"id": "ic3febng",
			"maxSelect": 99,
			"maxSize": 26214400,
			"mimeTypes": [
				"image/png",
				"image/vnd.mozilla.apng",
				"image/jpeg",
				"image/webp"
			],
			"name": "gallery",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": [
				"150x150",
				"640x480",
				"1280x720",
				"1920x1080",
				"320x240"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(27, []byte(`{
			"hidden": false,
			"id": "j0f8jpnt",
			"maxSelect": 1,
			"maxSize": 26214400,
			"mimeTypes": [],
			"name": "schematic_file",
			"presentable": false,
			"protected": false,
			"required": true,
			"system": false,
			"thumbs": null,
			"type": "file"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ezzomjw4q1qibza")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(25, []byte(`{
			"hidden": false,
			"id": "ptuuoygx",
			"maxSelect": 1,
			"maxSize": 10485760,
			"mimeTypes": [
				"image/png",
				"image/vnd.mozilla.apng",
				"image/jpeg",
				"image/webp"
			],
			"name": "featured_image",
			"presentable": false,
			"protected": false,
			"required": true,
			"system": false,
			"thumbs": [
				"150x150",
				"1920x1080",
				"640x360",
				"320x180"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(26, []byte(`{
			"hidden": false,
			"id": "ic3febng",
			"maxSelect": 99,
			"maxSize": 10485760,
			"mimeTypes": [
				"image/png",
				"image/vnd.mozilla.apng",
				"image/jpeg",
				"image/webp"
			],
			"name": "gallery",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": [
				"150x150",
				"640x480",
				"1280x720",
				"1920x1080",
				"320x240"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(27, []byte(`{
			"hidden": false,
			"id": "j0f8jpnt",
			"maxSelect": 1,
			"maxSize": 10485760,
			"mimeTypes": null,
			"name": "schematic_file",
			"presentable": false,
			"protected": false,
			"required": true,
			"system": false,
			"thumbs": null,
			"type": "file"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
