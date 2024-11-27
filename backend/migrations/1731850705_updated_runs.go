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

		collection, err := dao.FindCollectionByNameOrId("avereijhevumc07")
		if err != nil {
			return err
		}

		// update
		edit_status := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "5puiuhea",
			"name": "status",
			"type": "select",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"started",
					"completed",
					"interrupted",
					"error",
					"internal_error"
				]
			}
		}`), edit_status); err != nil {
			return err
		}
		collection.Schema.AddField(edit_status)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("avereijhevumc07")
		if err != nil {
			return err
		}

		// update
		edit_status := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "5puiuhea",
			"name": "status",
			"type": "select",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"started",
					"completed",
					"interrupted",
					"error"
				]
			}
		}`), edit_status); err != nil {
			return err
		}
		collection.Schema.AddField(edit_status)

		return dao.SaveCollection(collection)
	})
}
