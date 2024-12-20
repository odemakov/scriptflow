package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX `+"`"+`idx_n5BKghr`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`project`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_JDQvnOc`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`created`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_XUNSfgUfpE`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`active`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("text2560465762")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX `+"`"+`idx_r9TU1e1`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`node`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_n5BKghr`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`project`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_JDQvnOc`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`created`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_RcgWYwa`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`updated`+"`"+`)",
				"CREATE UNIQUE INDEX `+"`"+`idx_H4ZW9xeTZ8`+"`"+` ON `+"`"+`tasks`+"`"+` (\n  `+"`"+`slug`+"`"+`,\n  `+"`"+`project`+"`"+`\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text2560465762",
			"max": 64,
			"min": 6,
			"name": "slug",
			"pattern": "^[a-z0-9][a-z0-9\\-]+[a-z0-9]$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
