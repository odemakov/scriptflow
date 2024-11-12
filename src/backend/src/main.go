package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/desops/sshpool"
	"github.com/go-co-op/gocron"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"golang.org/x/crypto/ssh"
)

// define tasks collection name constant
const CollectionTasks = "tasks"
const CollectionRuns = "runs"
const CollectionNodes = "nodes"

type ScriptService struct {
    app       *pocketbase.PocketBase
    scheduler *gocron.Scheduler
}

func NewScriptService(app *pocketbase.PocketBase) *ScriptService {
    return &ScriptService{
        app:       app,
        scheduler: gocron.NewScheduler(time.UTC),
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

            _, err := s.scheduler.Tag(task.Id).SingletonMode().Cron(task.GetString("schedule")).Do(s.runTask, task, node)
            if err != nil {
                log.Printf("Failed to schedule task: %v", err)
            }
        }
    }
}

func (s *ScriptService) GetNodeConnection(node *models.Record) (*ssh.Session, error) {
    privateKey := node.GetString("private_key")
    /*  */signer, err := GetPrivateKey(&privateKey)
    if err != nil {
        return nil, err
    }

    config := &ssh.ClientConfig{
		User: node.GetString("username"),
		Auth: []ssh.AuthMethod{
            ssh.PublicKeys(signer),
		},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

    pool := sshpool.New(config, nil)
    session, err := pool.Get(node.GetString("host"))
    if err != nil {
        return nil, err
    }
    defer session.Put() // important: this returns it to the pool

    return session.Session, nil
}

// run task on node
func (s *ScriptService) runTask(task *models.Record, node *models.Record) {
    // find run collection
    runCollection, err := s.app.Dao().FindCollectionByNameOrId(CollectionRuns)
    if err != nil {
        log.Printf("Failed to find collection 'run': %v", err)
        return
    }

    // create `run` record
    run := models.NewRecord(runCollection)
    run.Set("task", task.Id)

    // connect to node
    session, err := s.GetNodeConnection(node)
    if err != nil {
        log.Printf("Failed get connection to node %s: %v", node.GetString("host"), err)
        run.Set("stderr", err.Error())
        run.Set("exit_code", 255)
        if err := s.app.Dao().SaveRecord(run); err != nil {
            log.Printf("Failed to save run log: %v", err)
        }
        return
    }

    log.Printf("Running command '%s' on node '%s'", task.GetString("command"), node.GetString("host"))
    // fmt.Fprintln(os.Stderr, "Random error")
    var stdout, stderr bytes.Buffer
    session.Stdout = &stdout
    session.Stderr = &stderr
    exitCode := 0

    if err := session.Run(task.GetString("command")); err != nil {
        ee, ok := err.(*ssh.ExitError)
        if ok {
            exitCode = ee.ExitStatus()
            fmt.Fprintf(os.Stderr, "remote command exit status %d\n", exitCode)
        } else {
            exitCode = 255
        }
    }

    run.Set("stdout", stdout.String())
    run.Set("stderr", stderr.String())
    run.Set("exit_code", exitCode)
    if err := s.app.Dao().SaveRecord(run); err != nil {
        log.Printf("Failed to save run log: %v", err)
    }
}

func GetPrivateKey(privateKey *string) (ssh.Signer, error) {
    if *privateKey == "" {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            return nil, err
        }
        keyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
        privateKey = &keyPath
    }
    key, err := os.ReadFile(*privateKey)
    if err != nil {
        return nil, err
    }
    signer, err := ssh.ParsePrivateKey(key)
    if err != nil {
        return nil, err
    }
    return signer, nil
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

    scriptService := NewScriptService(app)
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
