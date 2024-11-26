package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
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
			"CREATE INDEX ` + "`" + `idx_n5BKghr` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `project` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_JDQvnOc` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `created` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_RcgWYwa` + "`" + ` ON ` + "`" + `tasks` + "`" + ` (` + "`" + `updated` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
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

		return dao.SaveCollection(collection)
	})
}
