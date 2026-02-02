package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("tasks")
		if err != nil {
			return err
		}

		// Add consecutive_failure_count field
		collection.Fields.Add(&core.NumberField{
			Name:     "consecutive_failure_count",
			Min:      func() *float64 { v := 0.0; return &v }(),
			Required: false,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		// Revert: remove consecutive_failure_count field
		collection, err := app.FindCollectionByNameOrId("tasks")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("consecutive_failure_count")

		return app.Save(collection)
	})
}
