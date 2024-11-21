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
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

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
	})
}