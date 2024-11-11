/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("dzuidcfogskfz40")

  // add
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "bwujkkf4",
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
  }))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("dzuidcfogskfz40")

  // remove
  collection.schema.removeField("bwujkkf4")

  return dao.saveCollection(collection)
})
