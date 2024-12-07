package main

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/odemakov/sshrun"
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
	// get home directory of current user
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to get home directory: %v", err)
	}

	runCfg := &sshrun.RunConfig{
		DefaultPrivateKey: filepath.Join(homeDir, ".ssh", "id_rsa"),
	}
	sshPool := sshrun.NewPool(runCfg)

	sf := NewScriptFlow(app, sshPool)
	if err := sf.Start(); err != nil {
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

		// schedule JobCheckNodeStatus task to run every 30 seconds
		if _, err := sf.scheduler.Tag(SystemTask).Tag(JobCheckNodeStatus).SingletonMode().Every(30).Seconds().Do(func() {
			go sf.JobCheckNodeStatus()
		}); err != nil {
			sf.app.Logger().Error("failed to schedule JobCheckNodeStatus", slog.Any("err", err))
		}

		// schedule JobSendNotofocations task to run every 30 seconds
		if _, err := sf.scheduler.Tag(SystemTask).Tag(JobSendNotifications).SingletonMode().Every(30).Seconds().Do(func() {
			go sf.JobSendNotifications()
		}); err != nil {
			sf.app.Logger().Error("failed to schedule JobSendNotifications", slog.Any("err", err))
		}

		// schedule JobRemoveOutdatedLogs task
		if _, err := sf.scheduler.Tag(SystemTask).Tag(JobRemoveOutdatedLogs).SingletonMode().Cron("10 0 * * *").Do(func() {
			go sf.JobRemoveOutdatedLogs()
		}); err != nil {
			sf.app.Logger().Error("failed to schedule JobRemoveOutdatedLogs", slog.Any("err", err))
		}

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
			_ = sf.scheduler.RemoveByTag(e.Record.Id)
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
		e.Router.GET("/api/scriptflow/{projectId}/task/{taskId}/log-ws", sf.ApiTaskLogWebSocket)
		e.Router.GET("/api/scriptflow/{projectId}/run/{runId}/log", sf.ApiRunLog).Bind(apis.RequireAuth())
		e.Router.GET("/api/scriptflow/stats", sf.ApiScriptFlowStats).Bind(apis.RequireAuth())
		return e.Next()
	})
}
