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

		// remove
		collection.Schema.RemoveField("06fqp4vl")

		// update
		edit_project := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "zhrs29hb",
			"name": "project",
			"type": "relation",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"collectionId": "g42xf59f9op4szt",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": null,
				"displayFields": null
			}
		}`), edit_project); err != nil {
			return err
		}
		collection.Schema.AddField(edit_project)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

		// add
		del_singleton := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "06fqp4vl",
			"name": "singleton",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), del_singleton); err != nil {
			return err
		}
		collection.Schema.AddField(del_singleton)

		// update
		edit_project := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "zhrs29hb",
			"name": "project",
			"type": "relation",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"collectionId": "g42xf59f9op4szt",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), edit_project); err != nil {
			return err
		}
		collection.Schema.AddField(edit_project)

		return dao.SaveCollection(collection)
	})
}
