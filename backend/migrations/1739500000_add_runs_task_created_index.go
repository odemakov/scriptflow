package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		_, err := app.DB().NewQuery("CREATE INDEX IF NOT EXISTS idx_runs_task_created ON runs(task, created)").Execute()
		return err
	}, func(app core.App) error {
		_, err := app.DB().NewQuery("DROP INDEX IF EXISTS idx_runs_task_created").Execute()
		return err
	})
}
