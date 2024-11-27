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

		collection, err := dao.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

		// update
		edit_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "8bcsdbk7",
			"name": "name",
			"type": "text",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"min": 10,
				"max": 255,
				"pattern": ""
			}
		}`), edit_name); err != nil {
			return err
		}
		collection.Schema.AddField(edit_name)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

		// update
		edit_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "8bcsdbk7",
			"name": "name",
			"type": "text",
			"required": true,
			"presentable": true,
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
