package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Disable public self-registration; only superusers can create accounts
		collection.CreateRule = nil

		return app.Save(collection)
	}, func(app core.App) error {
		// Revert: re-enable public self-registration
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		emptyRule := ""
		collection.CreateRule = &emptyRule

		return app.Save(collection)
	})
}
