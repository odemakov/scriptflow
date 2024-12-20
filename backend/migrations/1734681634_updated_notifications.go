package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_2301922722")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": true,
			"collectionId": "pbc_3980638064",
			"hidden": false,
			"id": "relation2747688147",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "subscription",
			"presentable": true,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": true,
			"collectionId": "avereijhevumc07",
			"hidden": false,
			"id": "relation1349952704",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "run",
			"presentable": true,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_2301922722")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_3980638064",
			"hidden": false,
			"id": "relation2747688147",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "subscription",
			"presentable": true,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "avereijhevumc07",
			"hidden": false,
			"id": "relation1349952704",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "run",
			"presentable": true,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
