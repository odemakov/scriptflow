/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("dzuidcfogskfz40")

  // remove
  collection.schema.removeField("aj05cdsh")

  // add
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "5koc97yi",
    "name": "node",
    "type": "relation",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "collectionId": "4hznt7rq94fwfjb",
      "cascadeDelete": false,
      "minSelect": null,
      "maxSelect": null,
      "displayFields": null
    }
  }))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("dzuidcfogskfz40")

  // add
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "aj05cdsh",
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
  }))

  // remove
  collection.schema.removeField("5koc97yi")

  return dao.saveCollection(collection)
})
