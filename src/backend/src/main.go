package main

import (
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/odemakov/sshrun"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"

	//"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

const (
	CollectionTasks   = "tasks"
	CollectionRuns    = "runs"
	CollectionNodes   = "nodes"
	NodeStatusOnline  = "online"
	NodeStatusOffline = "offline"
	SchedulePeriod    = 60 // max delay in seconds for tasks with @every schedule
	logSeparator      = "[%s] [scriptflow] run %s"
)

const (
	RunStatusStarted       = "started"
	RunStatusError         = "error"
	RunStatusCompleted     = "completed"
	RunStatusInterrupted   = "interrupted"
	RunStatusInternalError = "internal_error"
)

type ScriptService struct {
	app       *pocketbase.PocketBase
	scheduler *gocron.Scheduler
	pool      *sshrun.Pool
	lock      sync.Mutex
}

// func convert task to string for logs
func NewScriptService(app *pocketbase.PocketBase, pool *sshrun.Pool) *ScriptService {
	return &ScriptService{
		app:       app,
		scheduler: gocron.NewScheduler(time.UTC),
		pool:      pool,
	}
}

func (s *ScriptService) Start() {
	s.scheduler.StartAsync()
}

func (s *ScriptService) scheduleTask(task *models.Record) {
	// Acquire lock to ensure scheduler access is thread-safe
	s.lock.Lock()
	defer s.lock.Unlock()

	// remove existing task
	s.scheduler.RemoveByTag(task.Id)

	// schedule new task if active
	if task.GetBool("active") {
		// find task nodes
		node, err := s.app.Dao().FindRecordById(CollectionNodes, task.GetString("node"))
		if err != nil {
			s.app.Logger().Error("failed to find node", taskAttrs(task))
			return
		}

		// if task.schedule begins with @
		if strings.HasPrefix(task.GetString("schedule"), "@") {
			// for @every 1m schedule task would run every minute from now
			// we add random delay here to avoid running all tasks at the same time
			time.Sleep(time.Duration(time.Second * time.Duration(rand.Intn(SchedulePeriod))))
		}

		// log scheduled task with task and node details
		s.app.Logger().Info("schedule task", nodeAttrs(node), taskAttrs(task))
		_, err = s.scheduler.Tag(task.Id).
			SingletonMode().
			Cron(task.GetString("schedule")).
			Do(s.runTask, task.Id, node.Id)
		if err != nil {
			s.app.Logger().Error("failed to schedule task", taskAttrs(task), nodeAttrs(node), slog.Any("error", err))
		}
	}
}

// run scheduled task
func (s *ScriptService) runTask(taskId string, nodeId string) {
	task, node, err := s.findNodeAndTaskToRun(taskId, nodeId)
	if err != nil {
		s.app.Logger().Error("failed to find node or task", slog.Any("error", err))
		return
	}

	// Create new run record
	run, err := s.createRunRecord(task, node)
	if err != nil {
		s.app.Logger().Error("failed to create record", slog.Any("error", err))
		return
	}

	// Create and open log file
	logFile, err := createLogFile(s.app, taskId)
	if err != nil {
		s.app.Logger().Error("Log file error", slog.Any("error", err))
		return
	}
	defer logFile.Close()

	// Configure SSH and run the command
	sshCfg := &sshrun.SSHConfig{
		User: node.GetString("username"),
		Host: node.GetString("host"),
	}

	// Execute command and process output
	s.app.Logger().Info("execute task", taskAttrs(task), nodeAttrs(node))
	exitCode, err := s.executeCommand(sshCfg, task, run, logFile)
	if err != nil {
		switch e := err.(type) {
		case *ScriptFlowError:
			s.app.Logger().Error("ScriptFlow error", nodeAttrs(node), taskAttrs(task), slog.Any("error", err))
			run.Set("status", RunStatusError)
			run.Set("status", RunStatusInternalError)
		case *sshrun.SSHError:
			s.app.Logger().Error("SSH error", nodeAttrs(node), taskAttrs(task), slog.Any("error", err))
			run.Set("connection_error", e.Msg)
			run.Set("status", RunStatusInterrupted)
		case *sshrun.CommandError:
			s.app.Logger().Error("command error", nodeAttrs(node), taskAttrs(task), slog.Any("error", err))
			run.Set("status", RunStatusError)
			run.Set("exit_code", exitCode)
		default:
			s.app.Logger().Error("unhandled error", nodeAttrs(node), taskAttrs(task), slog.Any("error", err))
			run.Set("status", RunStatusError)
		}
		if err := s.app.Dao().SaveRecord(run); err != nil {
			s.app.Logger().Error("failed to save run record", slog.Any("error", err))
		}
	} else {
		// Update run record with completion status
		run.Set("exit_code", exitCode)
		run.Set("status", RunStatusCompleted)
		if err := s.app.Dao().SaveRecord(run); err != nil {
			s.app.Logger().Error("failed to save run record(completed)", slog.Any("error", err))
		}
	}
}

// return node attributes for logging
func nodeAttrs(node *models.Record) slog.Attr {
	return slog.Any("node", map[string]interface{}{
		"id":       node.Id,
		"host":     node.GetString("host"),
		"username": node.GetString("username"),
	})
}

// return task attributes for logging
func taskAttrs(task *models.Record) slog.Attr {
	return slog.Any("task", map[string]interface{}{
		"id":       task.Id,
		"name":     task.GetString("name"),
		"schedule": task.GetString("schedule"),
	})
}

// return corresponding node and task to run
// check that node is online and task is active
func (s *ScriptService) findNodeAndTaskToRun(taskId string, nodeId string) (*models.Record, *models.Record, error) {
	// Fetch node record
	node, err := s.app.Dao().FindRecordById(CollectionNodes, nodeId)
	if err != nil {
		return nil, nil, err
	}

	// Skip task if the node is offline
	if node.GetString("status") != NodeStatusOnline {
		return nil, nil, NewNodeStatusNotOnlineError()
	}

	// Fetch task record
	task, err := s.app.Dao().FindRecordById(CollectionTasks, taskId)
	if err != nil {
		return nil, nil, err
	}

	// Skip task if it is not active
	if !task.GetBool("active") {
		return nil, nil, NewTaskNotActiveError()
	}

	return task, node, nil
}

func (s *ScriptService) createRunRecord(task, node *models.Record) (*models.Record, error) {
	runCollection, err := s.app.Dao().FindCollectionByNameOrId(CollectionRuns)
	if err != nil {
		return nil, fmt.Errorf("unable to find collection '%s': %w", CollectionRuns, err)
	}

	run := models.NewRecord(runCollection)
	run.Set("task", task.Id)
	run.Set("command", task.GetString("command"))
	run.Set("host", node.GetString("host"))
	run.Set("status", RunStatusStarted)
	if err := s.app.Dao().SaveRecord(run); err != nil {
		return nil, fmt.Errorf("failed to save run record: %w", err)
	}

	return run, nil
}

func (s *ScriptService) executeCommand(sshCfg *sshrun.SSHConfig, task *models.Record, run *models.Record, logFile *os.File) (int, error) {
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
	return s.pool.RunCombined(
		sshCfg,
		task.GetString("command"),
		func(out string) {
			// prepend datetime, if needed
			if prependDatetime {
				out = fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), out)
			}
			// Write output to log file
			if _, err := logFile.WriteString(out); err != nil {
				s.app.Logger().Error("failed to write to log file", slog.Any("error", err))
			}
			// Flush output to ensure fsnotify detects the changes
			if err := logFile.Sync(); err != nil {
				s.app.Logger().Error("failed to sync log file", slog.Any("error", err))
			}
		},
	)
}

func findActiveTasks(dao *daos.Dao) ([]*models.Record, error) {
	query := dao.RecordQuery(CollectionTasks).
		AndWhere(dbx.HashExp{"active": true}).
		Limit(1000)

	records := []*models.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

func onBeforeServeScheduleActiveTasks(dao *daos.Dao, scriptService *ScriptService) {
	// find all active tasks
	tasks, err := findActiveTasks(dao)
	if err != nil {
		scriptService.app.Logger().Error("failed to find active tasks", slog.Any("error", err))
		return
	}
	// schedule tasks one by one
	for _, task := range tasks {
		go scriptService.scheduleTask(task)
	}
}

func scheduleTasks(app *pocketbase.PocketBase) {
	// get home directory of current user
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	runCfg := &sshrun.RunConfig{
		PrivateKey: filepath.Join(homeDir, ".ssh", "id_rsa"),
	}
	sshPool := sshrun.NewPool(runCfg)

	scriptService := NewScriptService(app, sshPool)
	scriptService.Start()

	// schedule system tasks
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// schedule NodeStatus task to run every 30 seconds
		app.Logger().Info("scheduling system tasks")
		scriptService.scheduler.Tag("system-task").SingletonMode().Every(30).Seconds().Do(func() {
			go jobNodeStatus(app.Dao(), sshPool, app.Logger())
		})
		return nil
	})

	// Schedule existing tasks
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		app.Logger().Info("scheduling user tasks")
		onBeforeServeScheduleActiveTasks(app.Dao(), scriptService)
		return nil
	})

	// Schedule new tasks
	app.OnRecordAfterCreateRequest().Add(func(e *core.RecordCreateEvent) error {
		if e.Record.Collection().Name == CollectionTasks {
			go scriptService.scheduleTask(e.Record)
		}
		return nil
	})

	// Update exsisitng task
	app.OnRecordAfterUpdateRequest().Add(func(e *core.RecordUpdateEvent) error {
		if e.Record.Collection().Name == CollectionTasks {
			go scriptService.scheduleTask(e.Record)
		}
		return nil
	})

	// Remove scheduled task
	app.OnRecordBeforeDeleteRequest().Add(func(e *core.RecordDeleteEvent) error {
		if e.Record.Collection().Name == CollectionTasks {
			scriptService.scheduler.RemoveByTag(e.Record.Id)
		}
		return nil
	})
}

func createLogsWebSockets(app *pocketbase.PocketBase) {
	app.Logger().Info("websocket")
	// Register WebSocket handler
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// WebSocket doesn't support HTTP headers(Authorizations), we will use query params instead
		e.Router.GET("/api/scriptflow/task/:taskId/log-ws", handleTaskLogWebSocket(app))
		e.Router.GET("/api/scriptflow/run/:runId/log", handleRunLog(app), apis.RequireAdminOrRecordAuth("users"))
		e.Router.GET("/api/scriptflow/stats", handleWebSocketStats(), apis.RequireAdminOrRecordAuth("users"))
		return nil
	})
}

func main() {
	app := pocketbase.New()

	// loosely check if it was executed using "go run"
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		// (the isGoRun check is to enable it only during development)
		Automigrate: isGoRun,
	})

	scheduleTasks(app)

	createLogsWebSockets(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
