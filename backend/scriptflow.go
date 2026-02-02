package main

import (
	"context"
	"errors"
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

	ctx, cancel := context.WithCancel(context.Background())

	return &ScriptFlow{
		app:            app,
		config:         config,
		configFilePath: configFilePath,
		sshPool:        sshPool,
		scheduler:      scheduler,
		locks:          &ScriptFlowLocks{},
		logsDir:        filepath.Join(app.DataDir(), "..", "sf_logs"),
		ctx:            ctx,
		cancelFunc:     cancel,
		activeJobs:     make(map[string]gocron.Job),
		activeRuns:     make(map[string]context.CancelFunc),
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

	// schedule JobReconcileJobs task to run every hour
	_, err = sf.scheduler.NewJob(
		gocron.CronJob("0 * * * *", false),
		gocron.NewTask(func() {
			go sf.reconcileJobs()
		}),
		gocron.WithTags(SystemTask, JobReconcileJobs),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		sf.app.Logger().Error("failed to schedule JobReconcileJobs", slog.Any("error", err))
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

	taskId := task.GetString("id")

	// Check if job already exists
	existingJob, jobExists := sf.getActiveJob(taskId)

	// If task is not active, remove the job if it exists
	if !task.GetBool("active") {
		if jobExists {
			if err := sf.scheduler.RemoveJob(existingJob.ID()); err != nil {
				sf.app.Logger().Error("failed to remove inactive task", taskAttrs(task), slog.Any("error", err))
			} else {
				sf.removeActiveJob(taskId)
				sf.app.Logger().Info("removed inactive task", taskAttrs(task))
			}
		}
		return
	}

	// Task is active - create job definition
	sf.app.Logger().Info("schedule task", taskAttrs(task))
	schedule := task.GetString("schedule")

	var jobDefinition gocron.JobDefinition

	// Parse schedule to create appropriate job definition
	if strings.HasPrefix(schedule, "@every ") {
		duration, parseErr := time.ParseDuration(schedule[7:])
		if parseErr != nil {
			sf.app.Logger().Error("failed to parse duration", taskAttrs(task), slog.Any("error", parseErr))
			return
		}
		// spread tasks by 10% of duration to avoid running them simultaneously
		min, max := durationMinMax(duration)
		jobDefinition = gocron.DurationRandomJob(min, max)
	} else {
		// Resolve Jenkins-style H notation if present
		resolvedSchedule, err := resolveHashedSchedule(schedule, taskId)
		if err != nil {
			sf.app.Logger().Error("invalid hashed schedule", taskAttrs(task), slog.Any("error", err))
			return
		}
		if resolvedSchedule != schedule {
			sf.app.Logger().Debug("resolved hashed schedule",
				slog.String("taskId", taskId),
				slog.String("original", schedule),
				slog.String("resolved", resolvedSchedule))
		}
		jobDefinition = gocron.CronJob(resolvedSchedule, false)
	}

	taskFunc := gocron.NewTask(sf.runTask, taskId)
	jobOptions := []gocron.JobOption{
		gocron.WithTags(taskId),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	}

	if jobExists {
		// Update existing job
		updatedJob, err := sf.scheduler.Update(
			existingJob.ID(),
			jobDefinition,
			taskFunc,
			jobOptions...,
		)
		if err != nil {
			sf.app.Logger().Error("failed to update task", taskAttrs(task), slog.Any("error", err))
			return
		}
		sf.setActiveJob(taskId, updatedJob)
		sf.app.Logger().Info("updated existing task", taskAttrs(task))
	} else {
		// Create new job
		newJob, err := sf.scheduler.NewJob(
			jobDefinition,
			taskFunc,
			jobOptions...,
		)
		if err != nil {
			sf.app.Logger().Error("failed to schedule new task", taskAttrs(task), slog.Any("error", err))
			return
		}
		sf.setActiveJob(taskId, newJob)
		sf.app.Logger().Info("scheduled new task", taskAttrs(task))
	}
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

	// Create cancellable context for this run
	runCtx, runCancel := context.WithCancel(sf.ctx)
	sf.registerActiveRun(run.Id, runCancel)
	defer sf.unregisterActiveRun(run.Id)

	// Create and open log file
	logFile, err := sf.createLogFile(task.Id)
	if err != nil {
		sf.app.Logger().Error("Log file error", slog.Any("error", err))
		return
	}
	defer logFile.Close()

	// Execute command and process output
	sf.app.Logger().Info("execute task", taskAttrs(task), nodeAttrs(node))
	exitCode, err := sf.executeCommand(runCtx, nodeSSHConfig(node), task, run, logFile)
	if err != nil {
		// Check if context was cancelled (killed) first
		if errors.Is(err, context.Canceled) || runCtx.Err() == context.Canceled {
			sf.app.Logger().Info("task killed", nodeAttrs(node), taskAttrs(task))
			run.Set("status", RunStatusKilled)
		} else {
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

func (sf *ScriptFlow) executeCommand(ctx context.Context, sshCfg *sshrun.SSHConfig, task *core.Record, run *core.Record, logFile *os.File) (int, error) {
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
	return sf.sshPool.RunCombinedContext(
		ctx,
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
		Timeout:    10 * time.Second,
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
	// Serialize reload operations - only one reload at a time
	sf.reloadMutex.Lock()
	defer sf.reloadMutex.Unlock()

	sf.app.Logger().Info("Reload() method started")

	// Check if we have a config file to reload
	if sf.configFilePath == "" {
		sf.app.Logger().Info("No config file to reload - skipping")
		return nil
	}

	// Load new config from stored path
	newConfig, err := NewConfig(sf.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// Store old config for rollback
	sf.configMutex.Lock()
	oldConfig := sf.config
	sf.config = newConfig
	sf.configMutex.Unlock()

	// Update database from new config
	if err := sf.UpdateFromConfig(); err != nil {
		// Rollback on failure
		sf.configMutex.Lock()
		sf.config = oldConfig
		sf.configMutex.Unlock()
		return fmt.Errorf("failed to update from config, rolled back: %w", err)
	}

	// Reschedule active tasks with new config
	sf.scheduleActiveTasks()

	// Clean up any orphaned jobs immediately after reload
	sf.reconcileJobs()

	sf.app.Logger().Info("Configuration reloaded successfully")
	return nil
}

// Job management helper methods

func (sf *ScriptFlow) getActiveJob(taskId string) (gocron.Job, bool) {
	sf.jobsMutex.RLock()
	defer sf.jobsMutex.RUnlock()
	job, exists := sf.activeJobs[taskId]
	return job, exists
}

func (sf *ScriptFlow) setActiveJob(taskId string, job gocron.Job) {
	sf.jobsMutex.Lock()
	defer sf.jobsMutex.Unlock()
	sf.activeJobs[taskId] = job
}

func (sf *ScriptFlow) removeActiveJob(taskId string) {
	sf.jobsMutex.Lock()
	defer sf.jobsMutex.Unlock()
	delete(sf.activeJobs, taskId)
}

// Active runs management

func (sf *ScriptFlow) registerActiveRun(runId string, cancel context.CancelFunc) {
	sf.runsMutex.Lock()
	defer sf.runsMutex.Unlock()
	sf.activeRuns[runId] = cancel
}

func (sf *ScriptFlow) unregisterActiveRun(runId string) {
	sf.runsMutex.Lock()
	defer sf.runsMutex.Unlock()
	delete(sf.activeRuns, runId)
}

// KillRun cancels a running task by its run ID
func (sf *ScriptFlow) KillRun(runId string) error {
	sf.runsMutex.RLock()
	cancel, exists := sf.activeRuns[runId]
	sf.runsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("run %s is not active", runId)
	}

	sf.app.Logger().Info("killing run", slog.String("runId", runId))
	cancel()
	return nil
}

// UpdateTaskFailureCount updates the consecutive_failure_count field on a task
// when a run completes. Increments on error, resets to 0 on success.
func (sf *ScriptFlow) UpdateTaskFailureCount(run *core.Record) {
	status := run.GetString("status")

	// Only process terminal statuses (not "started")
	if status == RunStatusStarted {
		return
	}

	taskId := run.GetString("task")
	if taskId == "" {
		return
	}

	task, err := sf.app.FindRecordById(CollectionTasks, taskId)
	if err != nil {
		sf.app.Logger().Error("failed to find task for failure count update",
			slog.String("taskId", taskId),
			slog.Any("error", err))
		return
	}

	currentCount := task.GetInt("consecutive_failure_count")
	var newCount int

	switch status {
	case RunStatusCompleted:
		// Success - reset counter
		newCount = 0
	case RunStatusError, RunStatusInternalError:
		// Failure - increment counter
		newCount = currentCount + 1
	default:
		// Other statuses (interrupted, killed) - don't change counter
		return
	}

	// Only update if changed
	if newCount != currentCount {
		task.Set("consecutive_failure_count", newCount)
		if err := sf.app.Save(task); err != nil {
			sf.app.Logger().Error("failed to update task failure count",
				slog.String("taskId", taskId),
				slog.Any("error", err))
		}
	}
}
