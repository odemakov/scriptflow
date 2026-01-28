package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("runs")
		if err != nil {
			return err
		}

		// Find the status field and update its values
		for _, field := range collection.Fields {
			if field.GetName() == "status" {
				if selectField, ok := field.(*core.SelectField); ok {
					selectField.Values = []string{
						"started",
						"completed",
						"interrupted",
						"error",
						"internal_error",
						"killed",
					}
				}
				break
			}
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Revert: remove "killed" from status values
		collection, err := app.FindCollectionByNameOrId("runs")
		if err != nil {
			return err
		}

		for _, field := range collection.Fields {
			if field.GetName() == "status" {
				if selectField, ok := field.(*core.SelectField); ok {
					selectField.Values = []string{
						"started",
						"completed",
						"interrupted",
						"error",
						"internal_error",
					}
				}
				break
			}
		}

		return app.Save(collection)
	})
}
