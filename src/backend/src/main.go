package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/odemakov/sshrun"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

const (
    CollectionTasks = "tasks"
    CollectionRuns  = "runs"
    CollectionNodes = "nodes"
)

type ScriptService struct {
    app       *pocketbase.PocketBase
    scheduler *gocron.Scheduler
    pool      *sshrun.Pool
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

func (s *ScriptService) ScheduleTask(task *models.Record) {
    // remove existing task
    s.scheduler.RemoveByTag(task.Id)

    // schedule new task if active
    if task.GetBool("active") {
        // find task nodes
        nodes, err := FindNodes(s.app.Dao(), task)
        if err != nil {
            log.Printf("Failed to find node for task %v: %v", task, err)
            return
        }

        for _, node := range nodes {
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
    result, err := s.pool.Run(sshCfg, task.GetString("command"))
    log.Printf("Result: %v", result)
    log.Printf("Error: %v", err)
    exitCode := 0
    if err != nil {
        switch e := err.(type) {
        case *sshrun.SSHError:
            run.Set("connection_error", e.Msg)
            if err := s.app.Dao().SaveRecord(run); err != nil {
                log.Printf("Failed to save run log: %v", err)
            }
        case *sshrun.CommandError:
            exitCode = result.ExitCode
        default:
            log.Printf("Unknown error: %v", err)
            return
        }
    }
    run.Set("command", task.GetString("command"))
    run.Set("stdout", result.Stdout)
    run.Set("stderr", result.Stderr)
    run.Set("exit_code", exitCode)
    if err := s.app.Dao().SaveRecord(run); err != nil {
        log.Printf("Failed to save run log: %v", err)
    }
}

func FindActiveTasks(dao *daos.Dao) ([]*models.Record, error) {
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
func FindNodes(dao *daos.Dao, task *models.Record) ([]*models.Record, error) {
    ids := task.GetStringSlice("nodes")
    records, err := dao.FindRecordsByIds(CollectionNodes, ids)
    if err != nil {
        return nil, err
    }
    return records, nil
}

func main() {
    app := pocketbase.New()

    // get home directory of current user
    homeDir, err := os.UserHomeDir()
    if err != nil {
        log.Fatalf("Failed to get home directory: %v", err)
    }

    runCfg := &sshrun.RunConfig{
        Debug: true,
        PrivateKey: filepath.Join(homeDir, ".ssh", "id_rsa"),
    }
    sshPool := sshrun.NewPool(runCfg)

    scriptService := NewScriptService(app, sshPool)
    scriptService.Start()

    // Schedule existing tasks
    app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
        // find all active tasks
        tasks, err := FindActiveTasks(app.Dao())
        if err != nil {
            log.Printf("Failed to find active tasks: %v", err)
            return nil
        }
        log.Printf("Found %d active tasks", len(tasks))
        // schedule them one by one
        for _, task := range tasks {
            scriptService.ScheduleTask(task)
        }
        return nil
    })

    // Schedule new tasks
    app.OnRecordAfterCreateRequest().Add(func(e *core.RecordCreateEvent) error {
        if e.Record.Collection().Name == CollectionTasks {
            scriptService.ScheduleTask(e.Record)
        }
        return nil
    })

    // Update exsisitng task
    app.OnRecordAfterUpdateRequest().Add(func(e *core.RecordUpdateEvent) error {
        if e.Record.Collection().Name == CollectionTasks {
            scriptService.ScheduleTask(e.Record)
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
