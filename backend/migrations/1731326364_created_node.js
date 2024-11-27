/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "4hznt7rq94fwfjb",
    "created": "2024-11-11 11:59:24.760Z",
    "updated": "2024-11-11 11:59:24.760Z",
    "name": "node",
    "type": "base",
    "system": false,
    "schema": [
      {
        "system": false,
        "id": "c28ipjwu",
        "name": "host",
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
        "id": "dnstk9di",
        "name": "username",
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
  });

  return Dao(db).saveCollection(collection);
}, (db) => {
  const dao = new Dao(db);
  const collection = dao.findCollectionByNameOrId("4hznt7rq94fwfjb");

  return dao.deleteCollection(collection);
})
