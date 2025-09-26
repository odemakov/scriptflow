package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/odemakov/sshrun"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func NewScriptFlow(app *pocketbase.PocketBase, config *Config, configFilePath string) (*ScriptFlow, error) {
	// get home directory of current user
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	runCfg := &sshrun.RunConfig{
		DefaultPrivateKey: filepath.Join(homeDir, ".ssh", "id_rsa"),
	}
	sshPool := sshrun.NewPool(runCfg)

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	scheduler.Start()

	return &ScriptFlow{
		app:            app,
		config:         config,
		configFilePath: configFilePath,
		sshPool:        sshPool,
		scheduler:      scheduler,
		locks:          &ScriptFlowLocks{},
		logsDir:        filepath.Join(app.DataDir(), "..", "sf_logs"),
	}, nil
}

func (sf *ScriptFlow) Start() error {
	if err := os.MkdirAll(sf.logsDir, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func (sf *ScriptFlow) MarkAllRunningTasksAsInterrupted(errorMsg string) {
	// find all active runs
	_, err := sf.app.DB().Update(
		CollectionRuns,
		dbx.Params{
			"status":           RunStatusInterrupted,
			"connection_error": errorMsg,
		},
		dbx.HashExp{"status": RunStatusStarted},
	).Execute()

	if err != nil {
		sf.app.Logger().Error("failed to mark running tasks as interrupted", slog.Any("error", err))
	}
}

func (sf *ScriptFlow) scheduleSystemTasks() {
	// schedule JobCheckNodeStatus task to run every 30 seconds
	_, err := sf.scheduler.NewJob(
		gocron.DurationJob(30*time.Second),
		gocron.NewTask(func() {
			go sf.JobCheckNodeStatus()
		}),
		gocron.WithTags(SystemTask, JobCheckNodeStatus),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		sf.app.Logger().Error("failed to schedule JobCheckNodeStatus", slog.Any("error", err))
	}

	// schedule JobSendNotifications task to run every 30 seconds
	_, err = sf.scheduler.NewJob(
		gocron.DurationJob(30*time.Second),
		gocron.NewTask(func() {
			go sf.JobSendNotifications()
		}),
		gocron.WithTags(SystemTask, JobSendNotifications),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		sf.app.Logger().Error("failed to schedule JobSendNotifications", slog.Any("error", err))
	}

	// schedule JobRemoveOutdatedLogs task
	_, err = sf.scheduler.NewJob(
		gocron.CronJob("39 * * * *", false),
		gocron.NewTask(func() {
			go sf.JobRemoveOutdatedLogs()
		}),
		gocron.WithTags(SystemTask, JobRemoveOutdatedLogs),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		sf.app.Logger().Error("failed to schedule JobRemoveOutdatedLogs", slog.Any("error", err))
	}

	// schedule JobRemoveOutdatedRecords task
	_, err = sf.scheduler.NewJob(
		gocron.CronJob("39 1 * * *", false),
		gocron.NewTask(func() {
			go sf.JobRemoveOutdatedRecords()
		}),
		gocron.WithTags(SystemTask, JobRemoveOutdatedRecords),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		sf.app.Logger().Error("failed to schedule JobRemoveOutdatedRecords", slog.Any("error", err))
	}
}

func (sf *ScriptFlow) scheduleActiveTasks() {
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
		// do it in go routine with a delay to avoid schedule tasks simultaneously
		go sf.ScheduleTask(task)
	}
}

func (sf *ScriptFlow) ScheduleTask(task *core.Record) {
	// Acquire lock to ensure scheduler access is thread-safe
	sf.locks.scheduleTask.Lock()
	defer sf.locks.scheduleTask.Unlock()

	// remove existing task
	sf.scheduler.RemoveByTags(task.GetString("id"))

	// schedule new task if active
	if task.GetBool("active") {
		// log scheduled task with task and node details
		sf.app.Logger().Info("schedule task", taskAttrs(task))
		schedule := task.GetString("schedule")
		// if schedule starts with @every, it is a duration schedule
		if strings.HasPrefix(schedule, "@every ") {
			duration, err := time.ParseDuration(schedule[7:])
			if err != nil {
				sf.app.Logger().Error("failed to parse duration", taskAttrs(task), slog.Any("error", err))
				return
			}
			// spread tasks by 10% of duration to avoid running them simultaneously
			min, max := durationMinMax(duration)
			_, err = sf.scheduler.NewJob(
				gocron.DurationRandomJob(min, max),
				gocron.NewTask(sf.runTask, task.GetString("id")),
				gocron.WithTags(task.GetString("id")),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
			if err != nil {
				sf.app.Logger().Error("failed to schedule task", taskAttrs(task), slog.Any("error", err))
				return
			}
		} else {
			_, err := sf.scheduler.NewJob(
				gocron.CronJob(schedule, false),
				gocron.NewTask(sf.runTask, task.GetString("id")),
				gocron.WithTags(task.GetString("id")),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
			if err != nil {
				sf.app.Logger().Error("failed to schedule task", taskAttrs(task), slog.Any("error", err))
				return
			}
		}
	}
}

// function returns -10% and +10% of given duration
func durationMinMax(duration time.Duration) (time.Duration, time.Duration) {
	// calculate 10% spread from duration
	spread := duration / 10
	return duration - spread, duration + spread
}

// run scheduled task
func (sf *ScriptFlow) runTask(taskId string) {
	node, task, err := sf.findNodeAndTaskToRun(taskId)
	if err != nil {
		sf.app.Logger().Error("failed to find project, node or task", slog.Any("error", err))
		return
	}

	// Create new run record
	run, err := sf.createRunRecord(node, task)
	if err != nil {
		sf.app.Logger().Error("failed to create record", slog.Any("error", err))
		return
	}

	// Create and open log file
	logFile, err := sf.createLogFile(task.Id)
	if err != nil {
		sf.app.Logger().Error("Log file error", slog.Any("error", err))
		return
	}
	defer logFile.Close()

	// Execute command and process output
	sf.app.Logger().Info("execute task", taskAttrs(task), nodeAttrs(node))
	exitCode, err := sf.executeCommand(nodeSSHConfig(node), task, run, logFile)
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

// return corresponding project, node and task to run
// check that node is online and task is active
func (sf *ScriptFlow) findNodeAndTaskToRun(taskId string) (*core.Record, *core.Record, error) {
	// Fetch task record
	task, err := sf.app.FindRecordById(CollectionTasks, taskId)
	if err != nil {
		return nil, nil, err
	}

	// Fetch node record
	node, err := sf.app.FindRecordById(CollectionNodes, task.GetString("node"))
	if err != nil {
		return nil, nil, err
	}

	// Skip task if the node is offline
	if node.GetString("status") != NodeStatusOnline {
		return nil, nil, NewNodeStatusNotOnlineError()
	}

	// Skip task if it is not active
	if !task.GetBool("active") {
		return nil, nil, NewTaskNotActiveError()
	}

	return node, task, nil
}

func (sf *ScriptFlow) createRunRecord(node *core.Record, task *core.Record) (*core.Record, error) {
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
	return run, nil
}

func (sf *ScriptFlow) executeCommand(sshCfg *sshrun.SSHConfig, task *core.Record, run *core.Record, logFile *os.File) (int, error) {
	// add run mark to the log file
	runMark := fmt.Sprintf(
		LogSeparator,
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

func nodeSSHConfig(node *core.Record) *sshrun.SSHConfig {
	return &sshrun.SSHConfig{
		User:       node.GetString("username"),
		Host:       node.GetString("host"),
		PrivateKey: node.GetString("private_key"),
	}
}

// JobNodeStatus checks all the nodes and marks them as online or offline
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
			_, err := sf.sshPool.Run(nodeSSHConfig(node), "uptime", func(stdout string) {}, func(stderr string) {})
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

// retrieves all projects
func (sf *ScriptFlow) getProjects() ([]*core.Record, error) {
	projects, err := sf.app.FindAllRecords(CollectionProjects)
	if err != nil {
		sf.app.Logger().Error("failed to query project collection", slog.Any("error", err))
		return nil, err
	}
	return projects, nil
}

// extracts retention policy details for a project
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

func (sf *ScriptFlow) taskFileDate(fileName string) (time.Time, error) {
	// Ensure file name matches the format YYYYMMDD.log
	if len(fileName) != 12 || fileName[len(fileName)-4:] != ".log" {
		return time.Time{}, NewInvalidLogFileNameError()
	}

	// Parse date from filename
	dateStr := fileName[:8] // YYYYMMDD
	fileDate, err := time.Parse("20060102", dateStr)
	if err != nil {
		return time.Time{}, NewFailedParseDateFromLogFileNameError()
	}
	return fileDate, nil
}

// {sf.logsDir}/{taskId}
func (sf *ScriptFlow) taskLogRootDir(taskId string) string {
	return filepath.Join(
		sf.logsDir,
		taskId,
	)
}

// {taskLogRootDir}/{taskLogFileName}.log
func (sf *ScriptFlow) taskLogFilePathDate(taskId string, dateTime time.Time) string {
	fileName := TaskLogFileName(dateTime.UTC())
	return filepath.Join(
		sf.taskLogRootDir(taskId),
		fileName,
	)
}

// Helper function to get today's log file path
func (sf *ScriptFlow) taskTodayLogFilePath(taskId string) string {
	return sf.taskLogFilePathDate(taskId, time.Now())
}

func (sf *ScriptFlow) createLogFile(taskId string) (*os.File, error) {
	filePath := sf.taskTodayLogFilePath(taskId)
	logDir := filepath.Dir(filePath)
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		return nil, NewFailedCreateLogFileDirectoryError()
	}

	if _, err := os.Stat(filePath); err == nil {
		return os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	}
	return os.Create(filePath)
}

func (sf *ScriptFlow) Reload() error {
	sf.app.Logger().Info("Reload() method started")

	// Load new config from stored path
	if sf.configFilePath == "" {
		sf.app.Logger().Info("No config file to reload - skipping")
		return nil
	}

	newConfig, err := NewConfig(sf.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// Update config reference
	sf.config = newConfig

	// Update database from new config
	if err := sf.UpdateFromConfig(); err != nil {
		return fmt.Errorf("failed to update from config: %w", err)
	}

	sf.scheduleActiveTasks()

	return nil
}

