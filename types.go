package main

import (
	"log/slog"
	"sync"

	"github.com/go-co-op/gocron"
	"github.com/odemakov/sshrun"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	CollectionProjects = "projects"
	CollectionTasks    = "tasks"
	CollectionRuns     = "runs"
	CollectionNodes    = "nodes"
	NodeStatusOnline   = "online"
	NodeStatusOffline  = "offline"
	SchedulePeriod     = 60 // max delay in seconds for tasks with @every schedule
	LogSeparator       = "[%s] [scriptflow] run %s"
	LogsMaxDays        = 90
)

const (
	RunStatusStarted       = "started"
	RunStatusError         = "error"
	RunStatusCompleted     = "completed"
	RunStatusInterrupted   = "interrupted"
	RunStatusInternalError = "internal_error"
)

// ScriptFlowLocks encapsulates the locks for different tasks
type ScriptFlowLocks struct {
	scheduleTask          sync.Mutex
	jobCheckNodeStatus    sync.Mutex
	jobRemoveOutdatedLogs sync.Mutex
}

type ScriptFlow struct {
	app       *pocketbase.PocketBase
	scheduler *gocron.Scheduler
	sshPool   *sshrun.Pool
	locks     *ScriptFlowLocks
	logsDir   string
}

// type Node struct {
// 	Id 	 	 string `json:"id"`
// 	Host     string `json:"host"`
// 	Username string `json:"username"`
// 	Status   string `json:"status"`
// 	Created  types.DateTime `db:"created" json:"created"`
// 	Updated  types.DateTime `db:"updated" json:"updated"`
// }

// type Task struct {
// 	Id              string `json:"id"`
// 	Name            string `json:"name"`
// 	Command         string `json:"command"`
// 	Schedule        string `json:"schedule"`
// 	NodeId          string `json:"node"`
// 	ProjectId       string `json:"project"`
// 	Active          bool `json:"active"`
// 	PrependDateTime bool `json:"prependDateTime"`
// 	Created         types.DateTime `db:"created" json:"created"`
// 	Updated         types.DateTime `db:"updated" json:"updated"`
// }

// return node attributes for logging
func nodeAttrs(node *core.Record) slog.Attr {
	return slog.Any("node", map[string]interface{}{
		"id":       node.Id,
		"host":     node.GetString("host"),
		"username": node.GetString("username"),
	})
}

// return task attributes for logging
func taskAttrs(task *core.Record) slog.Attr {
	return slog.Any("task", map[string]interface{}{
		"id":       task.Id,
		"name":     task.GetString("name"),
		"schedule": task.GetString("schedule"),
	})
}

// return project attributes for logging
func projectAttrs(project *core.Record) slog.Attr {
	return slog.Any("task", map[string]interface{}{
		"id":     project.Id,
		"name":   project.GetString("name"),
		"config": project.GetString("config"),
	})
}

// ProjectConfig represents the JSON structure of the config field.
type ProjectConfig struct {
	LogsMaxDays *int `json:"logsMaxDays"`
}

// GetProjectConfig retrieves a specific attribute from the project's "config" JSON field.
// Returns the value of the attribute if found, or the defaultValue if the attribute is not present or invalid.
func GetProjectConfigAttr(project *core.Record, attr string, defaultValue interface{}) (interface{}, error) {
	// Retrieve the raw "config" field from the project
	var config ProjectConfig
	err := project.UnmarshalJSONField("config", &config)
	if err != nil {
		return defaultValue, nil // Return defaultValue if config cannot be parsed
	}

	// Handle specific attributes
	switch attr {
	case "logsMaxDays":
		if config.LogsMaxDays != nil {
			return *config.LogsMaxDays, nil // Dereference pointer to get the value
		}
		return defaultValue, nil // Use defaultValue if LogsMaxDays is nil
	default:
		return defaultValue, nil // Fallback for unsupported attributes
	}
}
