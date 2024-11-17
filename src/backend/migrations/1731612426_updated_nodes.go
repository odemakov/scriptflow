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

		collection, err := dao.FindCollectionByNameOrId("4hznt7rq94fwfjb")
		if err != nil {
			return err
		}

		// add
		new_status := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "sfntaiyv",
			"name": "status",
			"type": "select",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"online",
					"offline"
				]
			}
		}`), new_status); err != nil {
			return err
		}
		collection.Schema.AddField(new_status)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("4hznt7rq94fwfjb")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("sfntaiyv")

		return dao.SaveCollection(collection)
	})
}
