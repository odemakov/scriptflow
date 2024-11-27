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
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g42xf59f9op4szt")
		if err != nil {
			return err
		}

		// add
		new_config := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gtjx4fpk",
			"name": "config",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 2000000
			}
		}`), new_config); err != nil {
			return err
		}
		collection.Schema.AddField(new_config)

		// update
		edit_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "d0uy2qbi",
			"name": "name",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_name); err != nil {
			return err
		}
		collection.Schema.AddField(edit_name)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g42xf59f9op4szt")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("gtjx4fpk")

		// update
		edit_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "d0uy2qbi",
			"name": "name",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_name); err != nil {
			return err
		}
		collection.Schema.AddField(edit_name)

		return dao.SaveCollection(collection)
	})
}
