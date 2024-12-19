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
				"CREATE UNIQUE INDEX `+"`"+`idx_njIZ3pe`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`slug`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_r9TU1e1`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`node`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_n5BKghr`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`project`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_JDQvnOc`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`created`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_RcgWYwa`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`updated`+"`"+`)",
				"CREATE UNIQUE INDEX `+"`"+`idx_H4ZW9xeTZ8`+"`"+` ON `+"`"+`tasks`+"`"+` (\n  `+"`"+`slug`+"`"+`,\n  `+"`"+`project`+"`"+`\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("dzuidcfogskfz40")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_njIZ3pe`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`name`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_r9TU1e1`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`node`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_n5BKghr`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`project`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_JDQvnOc`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`created`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_RcgWYwa`+"`"+` ON `+"`"+`tasks`+"`"+` (`+"`"+`updated`+"`"+`)",
				"CREATE UNIQUE INDEX `+"`"+`idx_H4ZW9xeTZ8`+"`"+` ON `+"`"+`tasks`+"`"+` (\n  `+"`"+`slug`+"`"+`,\n  `+"`"+`project`+"`"+`\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
