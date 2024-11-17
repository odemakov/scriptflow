package main

import (
	"log/slog"

	"github.com/odemakov/sshrun"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// simple task that checks all the nodes and marks them as online or offline
func jobNodeStatus(dao *daos.Dao, sshPool *sshrun.Pool, logger *slog.Logger) {
	query := dao.RecordQuery(CollectionNodes).Limit(100)
	records := []*models.Record{}
	if err := query.All(&records); err != nil {
		logger.Error("failed to query nodes collection", slog.Any("error", err))
		return
	}

	// run 'uptime' command in goroutine on each node and mark node as online or offline
	for _, node := range records {
		logger.Debug("cleck node status", nodeAttrs(node))
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
				logger.Info("change node status", nodeAttrs(node), slog.String("old", oldStatus), slog.String("new", newStatus))
				node.Set("status", newStatus)
				if err := dao.SaveRecord(node); err != nil {
					logger.Error("failed to save node", slog.Any("error", err))
				}
				// close connection to the node if it is offline
				if newStatus == NodeStatusOffline {
					sshPool.Put(node.GetString("host"))
				}
			}
		}(node)
	}
}