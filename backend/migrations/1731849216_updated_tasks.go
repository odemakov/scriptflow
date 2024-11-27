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

		// add
		new_prepend_datetime := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "qomgsyqp",
			"name": "prepend_datetime",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_prepend_datetime); err != nil {
			return err
		}
		collection.Schema.AddField(new_prepend_datetime)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("qomgsyqp")

		return dao.SaveCollection(collection)
	})
}
