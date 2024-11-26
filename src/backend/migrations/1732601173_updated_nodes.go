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

		collection, err := dao.FindCollectionByNameOrId("4hznt7rq94fwfjb")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_qsgEuAI` + "`" + ` ON ` + "`" + `nodes` + "`" + ` (\n  ` + "`" + `host` + "`" + `,\n  ` + "`" + `username` + "`" + `\n)",
			"CREATE INDEX ` + "`" + `idx_lwtB3WG` + "`" + ` ON ` + "`" + `nodes` + "`" + ` (` + "`" + `status` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_9sUfDoR` + "`" + ` ON ` + "`" + `nodes` + "`" + ` (` + "`" + `created` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_lGmIMh7` + "`" + ` ON ` + "`" + `nodes` + "`" + ` (` + "`" + `updated` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("4hznt7rq94fwfjb")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_qsgEuAI` + "`" + ` ON ` + "`" + `nodes` + "`" + ` (\n  ` + "`" + `host` + "`" + `,\n  ` + "`" + `username` + "`" + `\n)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
