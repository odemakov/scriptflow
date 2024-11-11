package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"golang.org/x/crypto/ssh"
)

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
    _, err := s.scheduler.Cron(task.GetString("schedule")).Do(s.runTask, task)
    if err != nil {
        log.Printf("Failed to schedule task: %v", err)
    }
}

// create `run` struct
type Run struct {
    TaskId   string `json:"task_id"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
    ExitCode int    `json:"exit_code"`
}

func (s *ScriptService) runTask(task *models.Record) {
    // find task node
    node, err := FindNode(s.app.Dao(), task)
    if err != nil {
        log.Printf("Failed to find node for task %v: %v", task, err)
        return
    }

    // create `run` record
    runCollection, err := s.app.Dao().FindCollectionByNameOrId("run")
    if err != nil {
        log.Printf("Failed to find collection 'run': %v", err)
        return
    }
    run := models.NewRecord(runCollection)
    run.Set("task", task.Id)

    signer, err := GetPrivateKey(node)
    if err != nil {
        log.Printf("Failed to get private key: %v", err)
        run.Set("stderr", err.Error())
        if err := s.app.Dao().SaveRecord(run); err != nil {
            log.Printf("Failed to save run log: %v", err)
        }
        return
    }
    client, err := ssh.Dial("tcp", node.GetString("host"), &ssh.ClientConfig{
        User: node.GetString("username"),
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(signer),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    })
    if err != nil {
        log.Printf("Failed to connect to VM: %v", err)
        run.Set("stderr", err.Error())
        if err := s.app.Dao().SaveRecord(run); err != nil {
            log.Printf("Failed to save run log: %v", err)
        }
        return
    }
    defer client.Close()

    session, err := client.NewSession()
    if err != nil {
        log.Printf("Failed to create SSH session: %v", err)
        run.Set("stderr", err.Error())
        if err := s.app.Dao().SaveRecord(run); err != nil {
            log.Printf("Failed to save run log: %v", err)
        }
        return
    }
    defer session.Close()

    var stdout, stderr bytes.Buffer
    session.Stdout = &stdout
    session.Stderr = &stderr
    err = session.Run(task.GetString("command"))
    exitCode := 0
    if err != nil {
        if exitError, ok := err.(*ssh.ExitError); ok {
            exitCode = exitError.ExitStatus()
        } else {
            log.Printf("Failed to run command: %v", err)
            if err := s.app.Dao().SaveRecord(run); err != nil {
                log.Printf("Failed to save run log: %v", err)
            }
            return
        }
    }

    // run := models.NewRecord(runCollection)
    // run.Set("task_id", task.Id)
    run.Set("stdout", stdout.String())
    run.Set("stderr", stderr.String())
    run.Set("exit_code", exitCode)
    if err := s.app.Dao().SaveRecord(run); err != nil {
        log.Printf("Failed to save run log: %v", err)
    }
}

func GetPrivateKey(node *models.Record) (ssh.Signer, error) {
    keyPath := node.GetString("private_key")
    if keyPath == "" {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            return nil, err
        }
        keyPath = filepath.Join(homeDir, ".ssh", "id_rsa")
    }
    key, err := os.ReadFile(keyPath)
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
    query := dao.RecordQuery("tasks").
        AndWhere(dbx.HashExp{"active": true}).
        Limit(100)

    records := []*models.Record{}
    if err := query.All(&records); err != nil {
        return nil, err
    }
    return records, nil
}
func FindNode(dao *daos.Dao, task *models.Record) (*models.Record, error) {
    record, err := dao.FindRecordById("node", task.GetString("node"))
    if err != nil {
        return nil, err
    }
    return record, nil
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
        // schedule them
        for _, task := range tasks {
            log.Printf("Scheduling task: %v", task)
            scriptService.ScheduleTask(task)
        }

        return nil
    })

    // Schedule new tasks
    app.OnRecordAfterCreateRequest().Add(func(e *core.RecordCreateEvent) error {
        if e.Record.Collection().Name == "task" {
            scriptService.ScheduleTask(e.Record)
        }
        return nil
    })

    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
