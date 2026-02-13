package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"

	// track migrations
	_ "scriptflow/migrations"
)

// version of scriptflow
var Version = "(untracked)"

func loadConfig(configFilename string) *Config {
	if configFilename == "" {
		return nil
	}

	config, err := NewConfig(configFilename)
	if err != nil {
		log.Fatal("failed to open or read config file: ", err)
	}
	return config
}

func reloadCommand(app *pocketbase.PocketBase) error {
	// Find running scriptflow process and send SIGHUP signal
	pid, err := findScriptFlowPID(app.DataDir())
	if err != nil {
		return fmt.Errorf("failed to find running ScriptFlow process: %w", err)
	}

	// Send SIGHUP signal
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	if err := process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP to process %d: %w", pid, err)
	}

	fmt.Printf("Reload signal sent to ScriptFlow process (PID: %d)\n", pid)
	return nil
}

func createPIDFile(app *pocketbase.PocketBase) error {
	pidFile := filepath.Join(app.DataDir(), "scriptflow.pid")

	// Check if PID file already exists
	if _, err := os.Stat(pidFile); err == nil {
		// PID file exists, check if process is still running
		if existingPID, err := findScriptFlowPID(app.DataDir()); err == nil {
			return fmt.Errorf("ScriptFlow is already running with PID %d", existingPID)
		}
		// PID file exists but process is dead - remove stale PID file
		app.Logger().Info("Removing stale PID file", slog.String("path", pidFile))
		os.Remove(pidFile)
	}

	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
}

func findScriptFlowPID(dataDir string) (int, error) {
	pidFile := filepath.Join(dataDir, "scriptflow.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, fmt.Errorf("PID file not found or unreadable: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	// Verify process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		return 0, fmt.Errorf("process not found: %w", err)
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		return 0, fmt.Errorf("process %d is not running: %w", pid, err)
	}

	return pid, nil
}

func setupReloadSignalHandler(sf *ScriptFlow) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	go func() {
		defer signal.Stop(sigChan)
		for {
			select {
			case <-sf.ctx.Done():
				sf.app.Logger().Info("Signal handler shutting down")
				return
			case <-sigChan:
				if err := sf.Reload(); err != nil {
					sf.app.Logger().Error("failed to reload configuration", slog.Any("error", err))
				}
			}
		}
	}()
}

func main() {
	app := pocketbase.New()

	// redefine pocketbase's --version flag
	var showVersion bool
	app.RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version information")
	var configFilename string
	app.RootCmd.PersistentFlags().StringVar(&configFilename, "config", "", "set config file")
	app.RootCmd.ParseFlags(os.Args[1:])

	if showVersion {
		fmt.Printf("scriptflow version %s\n", Version)
		os.Exit(0)
	}

	// enable auto creation of migration files when making collection changes in the Dashboard
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: app.IsDev(),
	})

	// add reload command
	reloadCmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload configuration from file",
		Run: func(cmd *cobra.Command, args []string) {
			if err := reloadCommand(app); err != nil {
				log.Fatal("reload failed: ", err)
			}
		},
	}
	app.RootCmd.AddCommand(reloadCmd)

	config := loadConfig(configFilename)

	sf := initScriptFlow(app, config, configFilename)

	// Only setup signal handling and PID file for serve command
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Setup signal handling for reload
		setupReloadSignalHandler(sf)

		// Create PID file after serve starts
		if err := createPIDFile(app); err != nil {
			sf.app.Logger().Error("failed to create PID file", slog.Any("error", err))
			return fmt.Errorf("PID file creation failed: %w", err)
		} else {
			sf.app.Logger().Info("PID file created", slog.String("path", filepath.Join(app.DataDir(), "scriptflow.pid")))
		}

		return e.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

	if err := sf.Start(); err != nil {
		log.Fatal(err)
	}
}

func initScriptFlow(app *pocketbase.PocketBase, config *Config, configFilePath string) *ScriptFlow {
	sf, err := NewScriptFlow(app, config, configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	sf.app.Logger().Debug("setup scriptflow scheduler")
	sf.setupScheduler()

	sf.app.Logger().Debug("setup scriptflow API")
	sf.setupApi()

	sf.MountFs()

	return sf
}

func (sf *ScriptFlow) setupScheduler() {
	// schedule system tasks
	sf.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// update database from config file
		sf.configMutex.RLock()
		hasConfig := sf.config != nil
		sf.configMutex.RUnlock()

		if hasConfig {
			sf.UpdateFromConfig()
		}

		// mark all started tasks as interrupted, if any
		sf.MarkAllRunningTasksAsInterrupted("app-started")

		// Schedule system tasks
		sf.scheduleSystemTasks()

		// Schedule existing tasks, each tasks will be scheduled in their own goroutine
		sf.scheduleActiveTasks()

		// Start scheduler after all tasks are scheduled and PocketBase is fully ready
		sf.scheduler.Start()

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
		// Handle run status changes
		if e.Record.Collection().Name == CollectionRuns {
			go sf.ProcessRunNotification(e.Record)
			go sf.UpdateTaskFailureCount(e.Record)
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
			// Remove job using job tracking system
			if job, exists := sf.getActiveJob(e.Record.Id); exists {
				if err := sf.scheduler.RemoveJob(job.ID()); err != nil {
					sf.app.Logger().Error("failed to remove deleted task job",
						slog.String("taskId", e.Record.Id),
						slog.Any("error", err))
				} else {
					sf.removeActiveJob(e.Record.Id)
					sf.app.Logger().Info("removed deleted task job", slog.String("taskId", e.Record.Id))
				}
			}
			// it can take a while to remove all task logs, so we will do it in background
			go sf.RemoveTaskLogs(e.Record.Id)
		}
		return e.Next()
	})

	sf.app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		// Cancel context to clean up goroutines
		sf.cancelFunc()

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
	// Register API handlers
	sf.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// WebSocket doesn't support HTTP headers(Authorization), we will use query params instead
		e.Router.GET("/api/scriptflow/task/{taskId}/log-ws", sf.ApiTaskLogWebSocket)
		e.Router.GET("/api/scriptflow/run/{runId}/log", sf.ApiRunLog).Bind(apis.RequireAuth())
		e.Router.POST("/api/scriptflow/run/{runId}/kill", sf.ApiKillRun).Bind(apis.RequireAuth())
		e.Router.GET("/api/scriptflow/runs/latest", sf.ApiLatestRuns).Bind(apis.RequireAuth())
		e.Router.GET("/api/scriptflow/stats", sf.ApiScriptFlowStats).Bind(apis.RequireAuth())
		return e.Next()
	})
}
