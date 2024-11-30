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
)

const (
	RunStatusStarted       = "started"
	RunStatusError         = "error"
	RunStatusCompleted     = "completed"
	RunStatusInterrupted   = "interrupted"
	RunStatusInternalError = "internal_error"
)

type ScriptFlow struct {
	app       *pocketbase.PocketBase
	scheduler *gocron.Scheduler
	sshPool   *sshrun.Pool
	lock      sync.Mutex
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
