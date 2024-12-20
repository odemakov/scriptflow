package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("4hznt7rq94fwfjb")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_qsgEuAI`+"`"+` ON `+"`"+`nodes`+"`"+` (\n  `+"`"+`host`+"`"+`,\n  `+"`"+`username`+"`"+`\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("4hznt7rq94fwfjb")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_qsgEuAI`+"`"+` ON `+"`"+`nodes`+"`"+` (\n  `+"`"+`host`+"`"+`,\n  `+"`"+`username`+"`"+`\n)",
				"CREATE INDEX `+"`"+`idx_lwtB3WG`+"`"+` ON `+"`"+`nodes`+"`"+` (`+"`"+`status`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_9sUfDoR`+"`"+` ON `+"`"+`nodes`+"`"+` (`+"`"+`created`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_lGmIMh7`+"`"+` ON `+"`"+`nodes`+"`"+` (`+"`"+`updated`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
