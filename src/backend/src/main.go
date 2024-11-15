package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/odemakov/sshrun"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

const (
    CollectionTasks = "tasks"
    CollectionRuns  = "runs"
    CollectionNodes = "nodes"
    NodeStatusOnline = "online"
    NodeStatusOffline = "offline"
    SchedulePeriod = 60 // max delay in seconds for tasks with @every schedule
    LogsBasePath = "logs"
)

const (
    RunStatusStarted     = "started"
    RunStatusError       = "error"
    RunStatusCompleted   = "completed"
    RunStatusInterrupted = "interrupted"
)

type ScriptService struct {
    app       *pocketbase.PocketBase
    scheduler *gocron.Scheduler
    pool      *sshrun.Pool
    lock      sync.Mutex
}

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
        nodes, err := findNodes(s.app.Dao(), task)
        if err != nil {
            log.Printf("Failed to find node for task %v: %v", task, err)
            return
        }

        for _, node := range nodes {
            // if task.schedule begins with @
            if strings.HasPrefix(task.GetString("schedule"), "@") {
                // for @every 1m schedule task would run every minute from now
                // we add random delay here to avoid running all tasks at the same time
                time.Sleep(time.Duration(time.Second * time.Duration(rand.Intn(SchedulePeriod))))
            }

            log.Printf(
                "Scheduling task '%s' on node '%s@%s', command: '%s', cron: '%s'",
                task.GetString("name"),
                node.GetString("username"),
                node.GetString("host"),
                task.GetString("command"),
                task.GetString("schedule"),
            )

            if task.GetBool("singleton") {
                _, err := s.scheduler.Tag(task.Id).SingletonMode().Cron(task.GetString("schedule")).Do(s.runTask, task, node)
                if err != nil {
                    log.Printf("Failed to schedule task: %v", err)
                }
            } else {
                _, err := s.scheduler.Tag(task.Id).Cron(task.GetString("schedule")).Do(s.runTask, task, node)
                if err != nil {
                    log.Printf("Failed to schedule task: %v", err)
                }
            }
        }
    }
}

// run task on node
func (s *ScriptService) runTask(task *models.Record, node *models.Record) {
    // fetch node as status could have changed
    node, err := s.app.Dao().FindRecordById(CollectionNodes, node.Id)
    if err != nil {
        log.Printf("Failed to find node: %v", err)
        return
    }

    // skip running task if node isn't online 
    if node.GetString("status") != NodeStatusOnline {
        log.Printf("Node '%s' is not online, skip task %s", node.GetString("host"), task.Id)
        return
    }

    // find run collection
    runCollection, err := s.app.Dao().FindCollectionByNameOrId(CollectionRuns)
    if err != nil {
        log.Printf("Failed to find collection 'run': %v", err)
        return
    }

    // create run record
    run := models.NewRecord(runCollection)
    run.Set("task", task.Id)
    run.Set("command", task.GetString("command"))
    run.Set("host", node.GetString("host"))
    run.Set("status", RunStatusStarted)
    if err := s.app.Dao().SaveRecord(run); err != nil {
        log.Printf("Failed to save run log: %v", err)
        return
    }

    // create log directory if doesn't exist
    logDir := runLogPath(s.app, task, run)
    if _, err := os.Stat(logDir); os.IsNotExist(err) {
        if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
            log.Printf("Failed to create log directory: %v", err)
            return
        }
    }

    // we can't use blob storage here as it doesn't support append
    // I don't want to loose logs if the task fails
    filePath := filepath.Join(logDir, taskLogName(run))
    var logFile *os.File
    // if file exists, open it in append mode
    if _, err := os.Stat(filePath); err == nil {
        logFile, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
        if err != nil {
            log.Printf("Failed to open log file: %v", err)
            return
        }
        defer logFile.Close()
    } else {
        logFile, err = os.Create(filePath)
        if err != nil {
            log.Printf("Failed to create log file: %v", err)
            return
        }
        defer logFile.Close()
    }

    sshCfg := &sshrun.SSHConfig{
        User: node.GetString("username"),
        Host: node.GetString("host"),
    }

    log.Printf("Running command '%s' on node '%s'", task.GetString("command"), node.GetString("host"))
    // create run log and save it to the database
    exitCode, err := s.pool.RunCombined(
        sshCfg,
        task.GetString("command"),
        func(out string) {
            // append output to log file
            if _, err := logFile.WriteString(out); err != nil {
                log.Printf("Failed to write to log file: %v", err)
            }
        },
    )
    // close file
    if err := logFile.Close(); err != nil {
        log.Printf("Failed to close log file: %v", err)
    }

    if err != nil {
        switch e := err.(type) {
        case *sshrun.SSHError:
            run.Set("connection_error", e.Msg)
            if err := s.app.Dao().SaveRecord(run); err != nil {
                log.Printf("Failed to save run log: %v", err)
            }
        case *sshrun.CommandError:
            log.Printf("Command error: %v", err)
        default:
            log.Printf("Unknown error: %v", err)
            return
        }
    }
    run.Set("exit_code", exitCode)
    run.Set("status", RunStatusCompleted)
    if err := s.app.Dao().SaveRecord(run); err != nil {
        log.Printf("Failed to save run log: %v", err)
    }
}

func findActiveTasks(dao *daos.Dao) ([]*models.Record, error) {
    query := dao.RecordQuery(CollectionTasks).
        AndWhere(dbx.HashExp{"active": true}).
        Limit(100)

    records := []*models.Record{}
    if err := query.All(&records); err != nil {
        return nil, err
    }
    return records, nil
}

// find nodes for task
func findNodes(dao *daos.Dao, task *models.Record) ([]*models.Record, error) {
    ids := task.GetStringSlice("nodes")
    records, err := dao.FindRecordsByIds(CollectionNodes, ids)
    if err != nil {
        return nil, err
    }
    return records, nil
}

func onBeforeServeScheduleActiveTasks(dao *daos.Dao, scriptService *ScriptService) {
    // find all active tasks
    tasks, err := findActiveTasks(dao)
    if err != nil {
        log.Printf("Failed to find active tasks: %v", err)
        return
    }
    log.Printf("Found %d active tasks", len(tasks))
    // schedule them one by one
    for _, task := range tasks {
        go scriptService.scheduleTask(task)
    }
}

// function to generate file path based on run record
// data/logs/2024/01/05/{task.Id}
func runLogPath(app *pocketbase.PocketBase, task *models.Record, run *models.Record) string {
    created := run.GetCreated()
    year, month, day := created.Time().Date()

    // format: 2021/01/01
    return filepath.Join(
        app.DataDir(),
        LogsBasePath,
        strconv.Itoa(year),
        fmt.Sprintf("%02d", month),
        fmt.Sprintf("%02d", day),
        task.Id,
    )
}

// function to generate log file name based on task record
// {year}-{month}-{day}_{hour}-{minute}-{second}-{task.Id}.log
func taskLogName(run *models.Record) string {
    created := run.GetCreated()
    year, month, day := created.Time().Date()
    hour, minute, second := created.Time().Clock()
    return fmt.Sprintf(
        "%d%02d%02d-%02d%02d%02d.%s.log",
        year, month, day, hour, minute, second, run.Id,
    )
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

    // get home directory of current user
    homeDir, err := os.UserHomeDir()
    if err != nil {
        log.Fatalf("Failed to get home directory: %v", err)
    }

    runCfg := &sshrun.RunConfig{
        Debug: false,
        PrivateKey: filepath.Join(homeDir, ".ssh", "id_rsa"),
    }
    sshPool := sshrun.NewPool(runCfg)

    scriptService := NewScriptService(app, sshPool)
    scriptService.Start()

    // schedule system tasks
    app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
        // run NodeStatus task right away
        go jobNodeStatus(app.Dao(), sshPool) 

        // schedule NodeStatus task to run every minute
        log.Printf("Scheduling system tasks")
        scriptService.scheduler.Tag("system-task").SingletonMode().Every(30).Seconds().Do(func ()  {
           go jobNodeStatus(app.Dao(), sshPool) 
        })
        return nil
    })

    // Schedule existing tasks
    app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
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

    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
