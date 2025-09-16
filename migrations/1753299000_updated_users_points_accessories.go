package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			return err
		}

		// add points (number, onlyInt)
		if err := collection.Fields.AddMarshaledJSONAt(50, []byte(`{
			"hidden": false,
			"id": "number_user_points",
			"max": null,
			"min": null,
			"name": "points",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add accessories (text storing CSV of accessory keys)
		if err := collection.Fields.AddMarshaledJSONAt(51, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text_user_accessories",
			"max": 0,
			"min": 0,
			"name": "accessories",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			return err
		}
		collection.Fields.RemoveById("number_user_points")
		collection.Fields.RemoveById("text_user_accessories")
		return app.Save(collection)
	})
}
