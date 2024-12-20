package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("g42xf59f9op4szt")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_OryGj8POWy`+"`"+` ON `+"`"+`projects`+"`"+` (`+"`"+`slug`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("g42xf59f9op4szt")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_2XN4r5x`+"`"+` ON `+"`"+`projects`+"`"+` (`+"`"+`name`+"`"+`)",
				"CREATE UNIQUE INDEX `+"`"+`idx_OryGj8POWy`+"`"+` ON `+"`"+`projects`+"`"+` (`+"`"+`slug`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
