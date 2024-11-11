/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "db22rrnm8fr54p4",
    "created": "2024-11-11 09:47:44.816Z",
    "updated": "2024-11-11 09:47:44.816Z",
    "name": "pipeline",
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
  });

  return Dao(db).saveCollection(collection);
}, (db) => {
  const dao = new Dao(db);
  const collection = dao.findCollectionByNameOrId("db22rrnm8fr54p4");

  return dao.deleteCollection(collection);
})
