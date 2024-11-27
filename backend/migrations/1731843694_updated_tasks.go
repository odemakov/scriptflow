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

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_njIZ3pe` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `name` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_r9TU1e1` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `node` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_n5BKghr` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `project` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// update
		edit_node := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "5koc97yi",
			"name": "node",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "4hznt7rq94fwfjb",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), edit_node); err != nil {
			return err
		}
		collection.Schema.AddField(edit_node)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_njIZ3pe` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `name` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_r9TU1e1` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `nodes` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_n5BKghr` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `project` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// update
		edit_node := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "5koc97yi",
			"name": "nodes",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "4hznt7rq94fwfjb",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": null,
				"displayFields": null
			}
		}`), edit_node); err != nil {
			return err
		}
		collection.Schema.AddField(edit_node)

		return dao.SaveCollection(collection)
	})
}
