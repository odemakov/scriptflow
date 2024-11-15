package main

import (
	"log"
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
    // update node as status could have changed
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

    sshCfg := &sshrun.SSHConfig{
        User: node.GetString("username"),
        Host: node.GetString("host"),
    }

    log.Printf("Running command '%s' on node '%s'", task.GetString("command"), node.GetString("host"))
    // create buffer for stderr and stdout to fill them in Run'd callbacks
    stdOut, stdErr := "", ""
    exitCode, err := s.pool.Run(
        sshCfg,
        task.GetString("command"),
        func(stdout string) {
            stdOut += stdout
        },
        func(stderr string) {
            stdErr += stderr
        },
    )
    if err != nil {
        switch e := err.(type) {
        case *sshrun.SSHError:
            run.Set("connection_error", e.Msg)
            if err := s.app.Dao().SaveRecord(run); err != nil {
                log.Printf("Failed to save run log: %v", err)
            }
            s.pool.Put(node.GetString("host"))
        case *sshrun.CommandError:
            log.Printf("Command error: %v", err)
        default:
            log.Printf("Unknown error: %v", err)
            return
        }
    }
    run.Set("command", task.GetString("command"))
    run.Set("stdout", stdOut)
    run.Set("stderr", stdErr)
    run.Set("exit_code", exitCode)
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
