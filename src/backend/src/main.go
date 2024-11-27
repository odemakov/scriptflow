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

func initScheduler(sf *ScriptFlow) {
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
}

func initApi(sf *ScriptFlow) {
	sf.app.Logger().Info("ScriptFlow API")
	// Register WebSocket handler
	sf.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// WebSocket doesn't support HTTP headers(Authorization), we will use query params instead
		e.Router.GET("/api/scriptflow/task/{taskId}/log-ws", sf.ApiTaskLogWebSocket)
		e.Router.GET("/api/scriptflow/run/{runId}/log", sf.ApiRunLog).Bind(apis.RequireAuth())
		e.Router.GET("/api/scriptflow/stats", sf.ApiScriptFlowStats).Bind(apis.RequireAuth())

        // var upgrader = websocket.Upgrader{
        //     CheckOrigin: func(r *http.Request) bool {
        //         return true
        //     },
        // }

        // // register the websocket route
        // e.Router.GET("/api/scriptflow/ws", func(e *core.RequestEvent) error {
        //     conn, err := upgrader.Upgrade(e.Response, e.Request, nil)
        //     if err != nil {
        //         http.Error(e.Response, "Could not open websocket connection", http.StatusBadRequest)
        //         return err
        //     }
        //     defer conn.Close()

        //     for {
        //         // Read message from browser
        //         _, msg, err := conn.ReadMessage()
        //         if err != nil {
        //             break
        //         }

        //         // Write message back to browser
        //         if err = conn.WriteMessage(websocket.TextMessage, msg); err != nil {
        //             break
        //         }
        //     }
		// 	return e.Next()
        // })
		return e.Next()
	})
}


func initScriptFlow(app *pocketbase.PocketBase) {
	// get home directory of current user
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	runCfg := &sshrun.RunConfig{
		PrivateKey: filepath.Join(homeDir, ".ssh", "id_rsa"),
	}
	sshPool := sshrun.NewPool(runCfg)

	sf := NewScriptFlow(app, sshPool)
	sf.Start()

	initScheduler(sf)
	initApi(sf)
}

func main() {
	app := pocketbase.New()

	initScriptFlow(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
