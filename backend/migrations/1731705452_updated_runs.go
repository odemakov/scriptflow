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

		// remove
		collection.Schema.RemoveField("3bnjl0t3")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("avereijhevumc07")
		if err != nil {
			return err
		}

		// add
		del_output_file := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "3bnjl0t3",
			"name": "output_file",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), del_output_file); err != nil {
			return err
		}
		collection.Schema.AddField(del_output_file)

		return dao.SaveCollection(collection)
	})
}
