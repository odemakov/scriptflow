package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `[
			{
				"id": "_pb_users_auth_",
				"created": "2024-11-11 09:41:40.437Z",
				"updated": "2024-11-11 09:41:40.443Z",
				"name": "users",
				"type": "auth",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "users_name",
						"name": "name",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "users_avatar",
						"name": "avatar",
						"type": "file",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"mimeTypes": [
								"image/jpeg",
								"image/png",
								"image/svg+xml",
								"image/gif",
								"image/webp"
							],
							"thumbs": null,
							"maxSelect": 1,
							"maxSize": 5242880,
							"protected": false
						}
					}
				],
				"indexes": [],
				"listRule": "id = @request.auth.id",
				"viewRule": "id = @request.auth.id",
				"createRule": "",
				"updateRule": "id = @request.auth.id",
				"deleteRule": "id = @request.auth.id",
				"options": {
					"allowEmailAuth": true,
					"allowOAuth2Auth": true,
					"allowUsernameAuth": true,
					"exceptEmailDomains": null,
					"manageRule": null,
					"minPasswordLength": 8,
					"onlyEmailDomains": null,
					"onlyVerified": false,
					"requireEmail": false
				}
			},
			{
				"id": "dzuidcfogskfz40",
				"created": "2024-11-11 09:45:58.985Z",
				"updated": "2024-11-14 17:38:13.073Z",
				"name": "tasks",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "8bcsdbk7",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "bwujkkf4",
						"name": "command",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "gtj2rqfg",
						"name": "schedule",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "5koc97yi",
						"name": "nodes",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "4hznt7rq94fwfjb",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": null,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "mouungnm",
						"name": "active",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "06fqp4vl",
						"name": "singleton",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "zhrs29hb",
						"name": "project",
						"type": "relation",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"collectionId": "g42xf59f9op4szt",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "avereijhevumc07",
				"created": "2024-11-11 09:47:14.478Z",
				"updated": "2024-11-12 14:28:26.601Z",
				"name": "runs",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "iq7wk7ih",
						"name": "task",
						"type": "relation",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"collectionId": "dzuidcfogskfz40",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "bj2rir7o",
						"name": "connection_error",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "zz0fc2uh",
						"name": "stderr",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "vvjpumlf",
						"name": "stdout",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "oltov5yh",
						"name": "exit_code",
						"type": "number",
						"required": false,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": true
						}
					},
					{
						"system": false,
						"id": "nitnmf9s",
						"name": "command",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					}
				],
				"indexes": [],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "db22rrnm8fr54p4",
				"created": "2024-11-11 09:47:44.816Z",
				"updated": "2024-11-11 21:52:42.706Z",
				"name": "pipelines",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "2fp5bq8m",
						"name": "field",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "dzuidcfogskfz40",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": null,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "4hznt7rq94fwfjb",
				"created": "2024-11-11 11:59:24.760Z",
				"updated": "2024-11-14 15:47:47.502Z",
				"name": "nodes",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "c28ipjwu",
						"name": "host",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*)$"
						}
					},
					{
						"system": false,
						"id": "dnstk9di",
						"name": "username",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "ks7xzraa",
						"name": "private_key",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					}
				],
				"indexes": [],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "g42xf59f9op4szt",
				"created": "2024-11-14 17:37:39.165Z",
				"updated": "2024-11-14 17:37:39.165Z",
				"name": "projects",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "d0uy2qbi",
						"name": "name",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					}
				],
				"indexes": [],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			}
		]`

		collections := []*models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			return err
		}

		return daos.New(db).ImportCollections(collections, true, nil)
	}, func(db dbx.Builder) error {
		return nil
	})
}
