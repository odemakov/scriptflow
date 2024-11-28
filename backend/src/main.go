package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/odemakov/sshrun"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	initScriptFlow(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func initScriptFlow(app *pocketbase.PocketBase) {
	// get home directory of current user
	// TODO: this is not cross-platform and should be fixed.
	// Also service users often don't have home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to get home directory: %v", err)
	}

	runCfg := &sshrun.RunConfig{
		PrivateKey: filepath.Join(homeDir, ".ssh", "id_rsa"),
	}
	sshPool := sshrun.NewPool(runCfg)

	// register -logsDir parameter
	var logsDir string
	app.RootCmd.PersistentFlags().StringVar(&logsDir, "logsDir", "", "the directory with the Scriptflow logs")
	app.RootCmd.ParseFlags(os.Args[1:])

	sf := NewScriptFlow(app, sshPool, logsDir)
	sf.Start()

	sf.app.Logger().Info("setup scriptflow scheduler")
	sf.setupScheduler()

	sf.app.Logger().Info("init scriptflow API")
	sf.setupApi()

	sf.MountFs()
}

func(sf *ScriptFlow) setupScheduler() {
	// schedule system tasks
	sf.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// schedule NodeStatus task to run every 30 seconds
		sf.app.Logger().Info("scheduling system tasks")
		sf.scheduler.Tag("system-task").SingletonMode().Every(30).Seconds().Do(func() {
			go sf.JobNodeStatus()
		})
		return e.Next()
	})

	// Schedule existing tasks
	sf.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		sf.app.Logger().Info("scheduling user tasks")
		sf.scheduleActiveTasks()
		return e.Next()
	})

	// Schedule new tasks
	sf.app.OnRecordAfterCreateSuccess().BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Collection().Name == CollectionTasks {
			go sf.ScheduleTask(e.Record)
		}
		return e.Next()
	})

	// Update exsisitng task
	sf.app.OnRecordAfterUpdateSuccess().BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Collection().Name == CollectionTasks {
			go sf.ScheduleTask(e.Record)
		}
		return e.Next()
	})

	// Remove scheduled task
	sf.app.OnRecordAfterDeleteSuccess().BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Collection().Name == CollectionTasks {
			sf.scheduler.RemoveByTag(e.Record.Id)
		}
		return e.Next()
	})

	sf.app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		// Stop scheduler on app stop
		// In case of long running tasks it will wait for them to finish
		// Not sure I need this
		// sf.app.Logger().Info("stopping Scheduler")
		// sf.scheduler.Stop()

		// mark all running tasks as interrupted, if any
		sf.app.Logger().Info("marking all running tasks as interrupted")
		sf.MarkAllRunningTasksAsInterrupted()

		// Close all ssh connections thus terminate all running tasks, if any
		// sf.app.Logger().Info("stoping SSH Pool")
		// sf.sshPool.ClosePool()

		return e.Next()
	})
}

func(sf *ScriptFlow) setupApi() {
	// Register WebSocket handler
	sf.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// WebSocket doesn't support HTTP headers(Authorization), we will use query params instead
		e.Router.GET("/api/scriptflow/{projectId}/task/{taskId}/log-ws", sf.ApiTaskLogWebSocket)
		e.Router.GET("/api/scriptflow/{projectId}/run/{runId}/log", sf.ApiRunLog).Bind(apis.RequireAuth())
		e.Router.GET("/api/scriptflow/stats", sf.ApiScriptFlowStats).Bind(apis.RequireAuth())
		return e.Next()
	})
}