package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/odemakov/sshrun"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/exp/rand"
)

func NewScriptFlow(app *pocketbase.PocketBase, sshPool *sshrun.Pool) *ScriptFlow {
	return &ScriptFlow{
		app:       app,
		sshPool:   sshPool,
		scheduler: gocron.NewScheduler(time.UTC),
	}
}

func (sf *ScriptFlow) Start() {
	sf.scheduler.StartAsync()
}

func (sf *ScriptFlow) MarkAllRunningTasksAsInterrupted() {
	// find all active runs
	runs, err := sf.app.FindAllRecords(
		CollectionRuns,
		dbx.HashExp{"status": RunStatusStarted},
	)
	if err != nil {
		sf.app.Logger().Error("failed to find started runs", slog.Any("error", err))
		return
	}

	// mark them as interrupted
	for _, run := range runs {
		run.Set("status", RunStatusInterrupted)
		run.Set("error", "scriptflow interrupted")
		if err := sf.app.Save(run); err != nil {
			sf.app.Logger().Error("failed to save run", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow)scheduleActiveTasks() {
	// find all active tasks
	tasks, err := sf.app.FindAllRecords(
		CollectionTasks,
		dbx.HashExp{"active": true},
	)
	if err != nil {
		sf.app.Logger().Error("failed to find active tasks", slog.Any("error", err))
		return
	}

	// schedule tasks one by one
	for _, task := range tasks {
		go sf.ScheduleTask(task)
	}
}

func (sf *ScriptFlow) ScheduleTask(task *core.Record) {
	// Acquire lock to ensure scheduler access is thread-safe
	sf.lock.Lock()
	defer sf.lock.Unlock()

	// remove existing task
	sf.scheduler.RemoveByTag(task.GetString("id"))

	// schedule new task if active
	if task.GetBool("active") {
		// find task node
		node, err := sf.app.FindRecordById(CollectionNodes, task.GetString("node"))
		if err != nil {
			sf.app.Logger().Error("failed to find node", taskAttrs(task))
			return
		}

		// if task.schedule begins with @
		if strings.HasPrefix(task.GetString("scedule"), "@") {
			// for @every 1m schedule task would run every minute from now
			// we add random delay here to avoid running all tasks at the same time
			time.Sleep(time.Duration(time.Second * time.Duration(rand.Intn(SchedulePeriod))))
		}

		// log scheduled task with task and node details
		sf.app.Logger().Info("schedule task", nodeAttrs(node), taskAttrs(task))
		_, err = sf.scheduler.Tag(task.GetString("id")).
			SingletonMode().
			Cron(task.GetString("schedule")).
			Do(sf.runTask, task.GetString("id"), node.GetString("id"))
		if err != nil {
			sf.app.Logger().Error(
				"failed to schedule task",
				taskAttrs(task),
				nodeAttrs(node),
				slog.Any("error", err),
			)
		}
	}
}

// run scheduled task
func (sf *ScriptFlow) runTask(taskId string, nodeId string) {
	task, node, err := sf.findNodeAndTaskToRun(taskId, nodeId)
	if err != nil {
		sf.app.Logger().Error("failed to find node or task", slog.Any("error", err))
		return
	}

	// Create new run record
	run, err := sf.createRunRecord(task, node)
	if err != nil {
		sf.app.Logger().Error("failed to create record", slog.Any("error", err))
		return
	}

	// Create and open log file
	logFile, err := createLogFile(sf.app, taskId)
	if err != nil {
		sf.app.Logger().Error("Log file error", slog.Any("error", err))
		return
	}
	defer logFile.Close()

	// Configure SSH and run the command
	sshCfg := &sshrun.SSHConfig{
		User: node.GetString("username"),
		Host: node.GetString("host"),
	}

	// Execute command and process output
	sf.app.Logger().Info("execute task", taskAttrs(task), nodeAttrs(node))
	exitCode, err := sf.executeCommand(sshCfg, task, run, logFile)
	if err != nil {
		switch e := err.(type) {
		case *ScriptFlowError:
			sf.app.Logger().Error("ScriptFlow error", nodeAttrs(node), taskAttrs(task), slog.Any("error", err))
			run.Set("status", RunStatusInternalError)
		case *sshrun.SSHError:
			sf.app.Logger().Error("SSH error", nodeAttrs(node), taskAttrs(task), slog.Any("error", err))
			run.Set("connection_error", e.Msg)
			run.Set("status", RunStatusInterrupted)
		case *sshrun.CommandError:
			sf.app.Logger().Error("command error", nodeAttrs(node), taskAttrs(task), slog.Any("error", err))
			run.Set("status", RunStatusError)
			run.Set("exit_code", exitCode)
		default:
			sf.app.Logger().Error("unhandled error", nodeAttrs(node), taskAttrs(task), slog.Any("error", err))
			run.Set("status", RunStatusError)
		}
		if err := sf.app.Save(run); err != nil {
			sf.app.Logger().Error("failed to save run record", slog.Any("error", err))
		}
	} else {
		// Update run record with completion status
		run.Set("exit_code", exitCode)
		run.Set("status", RunStatusCompleted)
		if err := sf.app.Save(run); err != nil {
			sf.app.Logger().Error("failed to save run record(completed)", slog.Any("error", err))
		}
	}
}

// return corresponding node and task to run
// check that node is online and task is active
func (sf *ScriptFlow) findNodeAndTaskToRun(taskId string, nodeId string) (*core.Record, *core.Record, error) {
	// Fetch node record
	node, err := sf.app.FindRecordById(CollectionNodes, nodeId)
	if err != nil {
		return nil, nil, err
	}

	// Skip task if the node is offline
	if node.GetString("status") != NodeStatusOnline {
		return nil, nil, NewNodeStatusNotOnlineError()
	}

	// Fetch task record
	task, err := sf.app.FindRecordById(CollectionTasks, taskId)
	if err != nil {
		return nil, nil, err
	}

	// Skip task if it is not active
	if !task.GetBool("active") {
		return nil, nil, NewTaskNotActiveError()
	}

	return task, node, nil
}

func (sf *ScriptFlow) createRunRecord(task, node *core.Record) (*core.Record, error) {
	runCollection, err := sf.app.FindCollectionByNameOrId(CollectionRuns)
	if err != nil {
		return nil, fmt.Errorf("unable to find collection '%s': %w", CollectionRuns, err)
	}

	run := core.NewRecord(runCollection)
	run.Set("task", task.Id)
	run.Set("command", task.GetString("command"))
	run.Set("host", node.GetString("host"))
	run.Set("status", RunStatusStarted)
	if err := sf.app.Save(run); err != nil {
		return nil, fmt.Errorf("failed to save run record: %w", err)
	}

	// fetch created run record, otehrwise it won't update autoupdated fields
	returnRun, err := sf.app.FindRecordById(CollectionRuns, run.Id)
	if err != nil {
		return nil, fmt.Errorf("unable to find creted run: %w", err)
	}
	return returnRun, nil
}

func (sf *ScriptFlow) executeCommand(sshCfg *sshrun.SSHConfig, task *core.Record, run *core.Record, logFile *os.File) (int, error) {
	// add run mark to the log file
	runMark := fmt.Sprintf(
		logSeparator,
		time.Now().Format(time.RFC3339),
		run.Id,
	)
	if _, err := logFile.WriteString(runMark + "\n"); err != nil {
		return 0, &ScriptFlowError{"failed to write to log file"}
	}
	prependDatetime := task.GetBool("prepend_datetime")
	return sf.sshPool.RunCombined(
		sshCfg,
		task.GetString("command"),
		func(out string) {
			// prepend datetime, if needed
			if prependDatetime {
				out = fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), out)
			}
			// Write output to log file
			if _, err := logFile.WriteString(out); err != nil {
				sf.app.Logger().Error("failed to write to log file", slog.Any("error", err))
			}
			// Flush output to ensure fsnotify detects the changes
			if err := logFile.Sync(); err != nil {
				sf.app.Logger().Error("failed to sync log file", slog.Any("error", err))
			}
		},
	)
}

// simple task that checks all the nodes and marks them as online or offline
func (sf *ScriptFlow) JobNodeStatus() {
	nodes, err := sf.app.FindAllRecords(CollectionNodes)
	if err != nil {
		sf.app.Logger().Error("failed to query nodes collection", slog.Any("error", err))
		return
	}

	// run 'uptime' command in goroutine on each node and mark node as online or offline
	for _, node := range nodes {
		sf.app.Logger().Debug("check node status", nodeAttrs(node))
		go func(node *core.Record) {
			sshCfg := &sshrun.SSHConfig{
				User: node.GetString("username"),
				Host: node.GetString("host"),
			}
			oldStatus := node.GetString("status")
			var newStatus string
			// with empty callback functions, we just check if the command runs successfully
			_, err := sf.sshPool.Run(sshCfg, "uptime", func(stdout string) {}, func(stderr string) {})
			if err != nil {
				sf.app.Logger().Error("failed to run uptime command", nodeAttrs(node), slog.Any("error", err))
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

// simple task that remove outdated logs
func (sf *ScriptFlow) JobRemoveOutdatedLogs() {
}