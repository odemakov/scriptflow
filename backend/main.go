package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	// track migrations
	_ "scriptflow/migrations"
)

func main() {
	app := pocketbase.New()

	// enable auto creation of migration files when making collection changes in the Dashboard
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: app.IsDev(),
	})

	initScriptFlow(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func initScriptFlow(app *pocketbase.PocketBase) {
	sf, err := NewScriptFlow(app)
	if err != nil {
		log.Fatal(err)
	}

	sf.app.Logger().Info("setup scriptflow scheduler")
	sf.setupScheduler()

	sf.app.Logger().Info("setup scriptflow API")
	sf.setupApi()

	sf.MountFs()
}

func (sf *ScriptFlow) setupScheduler() {
	// schedule system tasks
	sf.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// mark all started tasks as interrupted, if any
		sf.MarkAllRunningTasksAsInterrupted("app-started")

		// Schedule system tasks
		sf.scheduleSystemTasks()

		// Schedule existing tasks, each tasks will be scheduled in their own goroutine
		sf.scheduleActiveTasks()

		return e.Next()
	})

	sf.app.OnRecordAfterCreateSuccess().BindFunc(func(e *core.RecordEvent) error {
		// Schedule new tasks
		if e.Record.Collection().Name == CollectionTasks {
			go sf.ScheduleTask(e.Record)
		}
		// init notification for run
		if e.Record.Collection().Name == CollectionRuns {
			go sf.ProcessRunNotification(e.Record)
		}

		return e.Next()
	})

	sf.app.OnRecordAfterUpdateSuccess().BindFunc(func(e *core.RecordEvent) error {
		// Update exsisitng task
		if e.Record.Collection().Name == CollectionTasks {
			go sf.ScheduleTask(e.Record)
		}
		// init notification for run
		if e.Record.Collection().Name == CollectionRuns {
			go sf.ProcessRunNotification(e.Record)
		}
		// Close node connection when node is updated, so that checkNodeStatus can attempt to reconnect with new params
		if e.Record.Collection().Name == CollectionNodes {
			sf.sshPool.Put(e.Record.GetString("host"))
		}

		return e.Next()
	})

	// Remove scheduled task
	sf.app.OnRecordAfterDeleteSuccess().BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Collection().Name == CollectionTasks {
			sf.scheduler.RemoveByTags(e.Record.Id)
			// it can take a while to remove all task logs, so we will do it in background
			go sf.RemoveTaskLogs(e.Record.Id)
		}
		return e.Next()
	})

	sf.app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		// make sure app is bootstrapped before marking tasks as interrupted
		if sf.app.IsBootstrapped() {
			// Stop scheduler on app stop
			// In case of long running tasks it will wait for them to finish
			// Not sure I need this
			// sf.app.Logger().Info("stopping Scheduler")
			// sf.scheduler.Stop()

			// mark all running tasks as interrupted, if any
			sf.app.Logger().Info("marking all running tasks as interrupted")
			sf.MarkAllRunningTasksAsInterrupted("app-terminated")

			// Close all ssh connections thus terminate all running tasks, if any
			// sf.app.Logger().Info("stoping SSH Pool")
			// sf.sshPool.ClosePool()
		}
		return e.Next()
	})
}

func (sf *ScriptFlow) setupApi() {
	// Register WebSocket handler
	sf.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// WebSocket doesn't support HTTP headers(Authorization), we will use query params instead
		e.Router.GET("/api/scriptflow/task/{taskId}/log-ws", sf.ApiTaskLogWebSocket)
		e.Router.GET("/api/scriptflow/task/{taskId}/run-once", sf.ApiTaskRun).Bind(apis.RequireAuth())
		e.Router.GET("/api/scriptflow/run/{runId}/log", sf.ApiRunLog).Bind(apis.RequireAuth())
		e.Router.GET("/api/scriptflow/stats", sf.ApiScriptFlowStats).Bind(apis.RequireAuth())
		return e.Next()
	})
}
