package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// JobCheckNodeStatus checks all the nodes and marks them as online or offline
func (sf *ScriptFlow) JobCheckNodeStatus() {
	nodes, err := sf.app.FindAllRecords(CollectionNodes)
	if err != nil {
		sf.app.Logger().Error("failed to query nodes collection", slog.Any("error", err))
		return
	}

	// run 'uptime' command in goroutine on each node and mark node as online or offline
	for _, node := range nodes {
		sf.app.Logger().Debug("check node status", nodeAttrs(node))
		go func(node *core.Record) {
			oldStatus := node.GetString("status")
			var newStatus string
			// with empty callback functions, we just check if the command runs successfully
			// use context with timeout to prevent goroutine leaks on unreachable nodes
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_, err := sf.sshPool.RunContext(ctx, nodeSSHConfig(node), "uptime", func(stdout string) {}, func(stderr string) {})
			if err != nil {
				sf.app.Logger().Error("failed to check node status", nodeAttrs(node), slog.Any("error", err))
				newStatus = NodeStatusOffline
			} else {
				newStatus = NodeStatusOnline
			}
			if oldStatus != newStatus {
				sf.app.Logger().Info(
					"change node status",
					slog.Any("node", node),
					slog.String("old", oldStatus),
					slog.String("new", newStatus),
				)
				query := sf.app.DB().Update(CollectionNodes, dbx.Params{"status": newStatus}, dbx.HashExp{"id": node.Id})
				result, err := query.Execute()
				if err != nil {
					sf.app.Logger().Error("failed to save node", slog.Any("error", err))
				} else {
					sf.app.Logger().Debug("update node status", slog.Any("result", result))
				}

				// close connection to the node if it is offline
				if newStatus == NodeStatusOffline {
					sf.sshPool.Put(node.GetString("host"))
				}
			}
		}(node)
	}
}

func (sf *ScriptFlow) RemoveTaskLogs(taskId string) error {
	logDir := sf.taskLogRootDir(taskId)
	if err := os.RemoveAll(logDir); err != nil {
		return fmt.Errorf("failed to remove task logs: %w", err)
	}
	return nil
}

func (sf *ScriptFlow) JobRemoveOutdatedLogs() {
	projects, err := sf.getProjects()
	if err != nil {
		return
	}

	for _, project := range projects {
		sf.app.Logger().Info("start remove outdated files for project", projectAttrs(project))

		cutoff, tasks, err := sf.getProjectRetentionDetails(project)
		if err != nil {
			continue
		}

		for _, task := range tasks {
			// Directory for log files
			logDir := sf.taskLogRootDir(task.Id)
			files, err := os.ReadDir(logDir)
			if err != nil {
				sf.app.Logger().Error("failed to read task log directory", taskAttrs(task), slog.Any("error", err))
				continue
			}

			// Iterate over files and remove outdated logs
			for _, file := range files {
				if file.IsDir() {
					continue
				}

				fileName := file.Name()
				fileDate, err := sf.taskFileDate(fileName)
				if err != nil {
					sf.app.Logger().Error("failed to parse log file name", slog.Any("fileName", fileName), slog.Any("error", err))
					continue
				}

				// Remove file if older than logsMaxDays
				if fileDate.Before(cutoff) {
					filePath := filepath.Join(logDir, fileName)
					err := os.Remove(filePath)
					if err != nil {
						sf.app.Logger().Error("failed to remove outdated log file", slog.String("filePath", filePath), slog.Any("error", err))
					} else {
						sf.app.Logger().Info("removed outdated log file", slog.String("filePath", filePath))
					}
				}
			}
		}
	}
}

func (sf *ScriptFlow) JobRemoveOutdatedRecords() {
	projects, err := sf.getProjects()
	if err != nil {
		return
	}

	for _, project := range projects {
		sf.app.Logger().Info("start remove outdated records for project", projectAttrs(project))

		cutoff, tasks, err := sf.getProjectRetentionDetails(project)
		if err != nil {
			continue
		}

		for _, task := range tasks {
			query := sf.app.DB().Delete(
				CollectionRuns,
				dbx.NewExp(
					"task = {:task} AND created < {:created}",
					dbx.Params{"task": task.Id, "created": cutoff},
				))
			result, err := query.Execute()
			if err != nil {
				sf.app.Logger().Error("failed to delete runs", slog.Any("error", err))
				continue
			}

			affected, _ := result.RowsAffected()
			if affected > 0 {
				sf.app.Logger().Info("deleted outdated run records",
					slog.Int64("count", affected),
					slog.String("taskId", task.Id),
					slog.Time("olderThan", cutoff),
				)
			}
		}
	}
}

// getProjects retrieves all projects
func (sf *ScriptFlow) getProjects() ([]*core.Record, error) {
	projects, err := sf.app.FindAllRecords(CollectionProjects)
	if err != nil {
		sf.app.Logger().Error("failed to query project collection", slog.Any("error", err))
		return nil, err
	}
	return projects, nil
}

// getProjectRetentionDetails extracts retention policy details for a project
func (sf *ScriptFlow) getProjectRetentionDetails(project *core.Record) (time.Time, []*core.Record, error) {
	logsMaxDays, err := GetCollectionConfigAttr(project, "logsMaxDays", LogsMaxDays)
	if err != nil {
		sf.app.Logger().Error("failed to get project's logsMaxDays attr", slog.Any("error", err))
		return time.Time{}, nil, err
	}

	// logsMaxDays is returned as an interface{}, so assert its type
	logsMaxDaysInt, ok := logsMaxDays.(int)
	if !ok {
		sf.app.Logger().Error("unexpected type for logsMaxDays, expected int, got: %T\n", logsMaxDays)
		return time.Time{}, nil, fmt.Errorf("unexpected type for logsMaxDays")
	}

	tasks, err := sf.app.FindAllRecords(CollectionTasks, dbx.HashExp{"project": project.Id})
	if err != nil {
		sf.app.Logger().Error("failed to query tasks collection", slog.Any("error", err))
		return time.Time{}, nil, err
	}

	// Calculate cutoff time, add one extra day
	cutoff := time.Now().AddDate(0, 0, -logsMaxDaysInt-1)

	return cutoff, tasks, nil
}

func (sf *ScriptFlow) JobSendNotifications() {
	// select 10 last notifications where sent is false, sort by created
	notifications, err := sf.app.FindRecordsByFilter(
		CollectionNotifications,
		"sent={:sent} && error_count<={:error_count}",
		"updated",
		1,
		0,
		dbx.Params{"sent": false, "error_count": SendMaxErrorCount},
	)
	if err != nil {
		sf.app.Logger().Error("failed to query notifications collection", slog.Any("error", err))
		return
	}

	for _, notification := range notifications {
		// retrieve run
		run, err := sf.app.FindRecordById(CollectionRuns, notification.GetString("run"))
		if err != nil {
			sf.app.Logger().Error("failed to find run", slog.Any("error", err))
			continue
		}
		// retrieve task
		task, err := sf.app.FindRecordById(CollectionTasks, run.GetString("task"))
		if err != nil {
			sf.app.Logger().Error("failed to find task", slog.Any("error", err))
			continue
		}
		// retrieve project
		project, err := sf.app.FindRecordById(CollectionProjects, task.GetString("project"))
		if err != nil {
			sf.app.Logger().Error("failed to find project", slog.Any("error", err))
			continue
		}
		// retrieve subscription
		subscription, err := sf.app.FindRecordById(CollectionSubscriptions, notification.GetString("subscription"))
		if err != nil {
			sf.app.Logger().Error("failed to find subscription", slog.Any("error", err))
			continue
		}
		// retrieve channel
		channel, err := sf.app.FindRecordById(CollectionChannels, subscription.GetString("channel"))
		if err != nil {
			sf.app.Logger().Error("failed to find channel", slog.Any("error", err))
			continue
		}
		// send notification
		err = sf.sendNotification(NotificationContext{
			Project:      project,
			Task:         task,
			Run:          run,
			Notification: notification,
			Subscription: subscription,
			Channel:      channel,
		})
		if err != nil {
			sf.app.Logger().Error("failed to send notification", slog.Any("error", err))
			// increment error counter
			notification.Set("error_count", notification.GetInt("error_count")+1)
			if err := sf.app.Save(notification); err != nil {
				sf.app.Logger().Error("failed to save notification", slog.Any("error", err))
			}
		} else {
			sf.app.Logger().Info("notification sent", slog.Any("notification", notification))
			// mark notification as sent
			notification.Set("sent", true)
			if err := sf.app.Save(notification); err != nil {
				sf.app.Logger().Error("failed to save notification", slog.Any("error", err))
			}
		}
	}
}
