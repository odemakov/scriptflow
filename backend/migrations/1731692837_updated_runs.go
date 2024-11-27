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
		collection.Schema.RemoveField("zz0fc2uh")

		// remove
		collection.Schema.RemoveField("vvjpumlf")

		// add
		new_host := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "nyblsvc0",
			"name": "host",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_host); err != nil {
			return err
		}
		collection.Schema.AddField(new_host)

		// add
		new_log := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "sw5sye8m",
			"name": "log",
			"type": "file",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"mimeTypes": [],
				"thumbs": [],
				"maxSelect": 1,
				"maxSize": 5242880,
				"protected": false
			}
		}`), new_log); err != nil {
			return err
		}
		collection.Schema.AddField(new_log)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("avereijhevumc07")
		if err != nil {
			return err
		}

		// add
		del_stderr := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "zz0fc2uh",
			"name": "stderr",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), del_stderr); err != nil {
			return err
		}
		collection.Schema.AddField(del_stderr)

		// add
		del_stdout := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "vvjpumlf",
			"name": "stdout",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), del_stdout); err != nil {
			return err
		}
		collection.Schema.AddField(del_stdout)

		// remove
		collection.Schema.RemoveField("nyblsvc0")

		// remove
		collection.Schema.RemoveField("sw5sye8m")

		return dao.SaveCollection(collection)
	})
}
