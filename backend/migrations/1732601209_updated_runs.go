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

		collection, err := dao.FindCollectionByNameOrId("avereijhevumc07")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX ` + "`" + `idx_RSsvRki` + "`" + ` ON ` + "`" + `runs` + "`" + ` (` + "`" + `task` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_5jtH1Wa` + "`" + ` ON ` + "`" + `runs` + "`" + ` (` + "`" + `exit_code` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_uWeeaKk` + "`" + ` ON ` + "`" + `runs` + "`" + ` (` + "`" + `status` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_1d5KcBV` + "`" + ` ON ` + "`" + `runs` + "`" + ` (` + "`" + `created` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_K4gdJ2m` + "`" + ` ON ` + "`" + `runs` + "`" + ` (` + "`" + `updated` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("avereijhevumc07")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX ` + "`" + `idx_RSsvRki` + "`" + ` ON ` + "`" + `runs` + "`" + ` (` + "`" + `task` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_5jtH1Wa` + "`" + ` ON ` + "`" + `runs` + "`" + ` (` + "`" + `exit_code` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
