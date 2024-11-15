package main

import (
	"log"

	"github.com/odemakov/sshrun"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// simple task that checks all the nodes and marks them as online or offline
func jobNodeStatus(dao *daos.Dao, sshPool *sshrun.Pool) {
	query := dao.RecordQuery(CollectionNodes).Limit(100)
	records := []*models.Record{}
	if err := query.All(&records); err != nil {
		log.Printf("Failed to query nodes collection: %v", err)
		return
	}

	// run 'uptime' command in goroutine on each node and mark node as online or offline
	for _, node := range records {
		go func(node *models.Record) {
			sshCfg := &sshrun.SSHConfig{
				User: node.GetString("username"),
				Host: node.GetString("host"),
			}
			oldStatus := node.GetString("status")
			var newStatus string
			// with empty callback functions, we just check if the command runs successfully
			_, err := sshPool.Run(sshCfg, "uptime", func(stdout string) {}, func(stderr string) {})
			if err != nil {
				newStatus = NodeStatusOffline
			} else {
				newStatus = NodeStatusOnline
			}
			if oldStatus != newStatus {
				log.Printf("Change node '%s' status: %s -> %s", node.GetString("host"), oldStatus, newStatus)
				node.Set("status", newStatus)
				if err := dao.SaveRecord(node); err != nil {
					log.Printf("Failed to save node status: %v", err)
				}
			}
		}(node)
	}
}