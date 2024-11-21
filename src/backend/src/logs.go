package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

const (
	LogsBasePath = "logs"
)

var (
	openWebSockets      int
	openWebSocketsMutex sync.Mutex
)

func handleTaskLogWebSocket(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskId := c.PathParam("taskId")

		// Upgrade HTTP connection to WebSocket
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Adjust as needed for security
			},
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upgrade WebSocket")
		}
		defer conn.Close()

		// Increment the counter
		openWebSocketsMutex.Lock()
		openWebSockets++
		openWebSocketsMutex.Unlock()

		defer func() {
			// Decrement the counter
			openWebSocketsMutex.Lock()
			openWebSockets--
			openWebSocketsMutex.Unlock()
		}()

		// Locate the log file
		logFilePath := taskTodayLogFilePath(app, taskId)
		app.Logger().Debug("TaskLogWebSocket handler", slog.String("file", logFilePath))
		if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
			conn.WriteMessage(websocket.TextMessage, []byte("Log file not found"))
			return nil
		}

		if err := sendLastLines(conn, logFilePath, 100); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to send last lines")
		}
		if err := watchFileChanges(conn, logFilePath, app); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to watch file changes")
		}
		return nil
	}
}

// function to retrieve run log by runId
// it extract task_id by makeing join quesry to databases
// finds log file by run created date and task id
// parse log file and return run log content
// one log file for task per day
// run logs separated by '[Datetime] ########## Run {run.id} ##########'
func handleRunLog(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		runId := c.PathParam("runId")

		// select run by run id
		run, err := app.Dao().FindRecordById(CollectionRuns, runId)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "Run not found")
		}

		// select task by run id
		task, err := app.Dao().FindRecordById(CollectionTasks, run.GetString("task"))
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "Task not found")
		}

		// get log file path
		logFilePath := taskLogFilePathDate(
			app,
			task.GetString("id"),
			run.GetDateTime("created").Time(),
		)
		logs, err := extractLogsForRun(logFilePath, runId)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// read log file
		// return {data: logs: []string}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"logs": logs,
		})
	}
}

func extractLogsForRun(logFilePath, runId string) ([]string, error) {
	//delimiterRegex := regexp.MustCompile(`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+\d{2}:\d{2}\] \[scriptflow\] run ([a-z0-9]{15})`)
	delimiterRegex := regexp.MustCompile(`^\[.*\] \[scriptflow\] run (\S+)$`)

	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %v: %v", logFilePath, err)
	}
	defer file.Close()

	var collecting bool
	var logs []string
	logCount := 0 // Counter for the number of lines in the logs slice
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Check if the line matches the delimiter pattern
		if matches := delimiterRegex.FindStringSubmatch(line); matches != nil {
			// Extract the runId from the delimiter line
			currentRunId := matches[1]

			// Determine if we should start or stop collecting logs
			if currentRunId == runId {
				collecting = true
				// Add the delimiter line, ensuring a rolling window
				if logCount == 10000 {
					logs = logs[1:] // Remove the first line to make space
					logCount--      // Decrement the counter
				}
				logs = append(logs, line)
				logCount++ // Increment the counter
			} else {
				// Stop collecting logs if the runId changes
				if collecting {
					break
				}
				collecting = false
			}
		} else if collecting {
			// Add the log line for the matching runId, ensuring a rolling window
			if logCount == 10000 {
				logs = logs[1:] // Remove the first line to make space
				logCount--      // Decrement the counter
			}
			logs = append(logs, line)
			logCount++ // Increment the counter
		}
	}
	return logs, nil
}

func handleWebSocketStats() echo.HandlerFunc {
	return func(c echo.Context) error {
		openWebSocketsMutex.Lock()
		count := openWebSockets
		openWebSocketsMutex.Unlock()

		return c.JSON(http.StatusOK, map[string]int{"openWebSockets": count})
	}
}

// Read and send the last N lines of the file
func sendLastLines(conn *websocket.Conn, filePath string, n int) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	lines, err := readLastLines(file, n)
	if err != nil {
		return err
	}

	for _, line := range lines {
		message := line + "\n"
		if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			return err
		}
	}
	return nil
}

// Watch for changes in the log file and send new content
func watchFileChanges(conn *websocket.Conn, filePath string, app *pocketbase.PocketBase) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(filePath); err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Start at the end of the file
	offset, err := file.Seek(0, os.SEEK_END)
	if err != nil {
		return err
	}
	app.Logger().Debug("watchFileChanges", slog.Int64("offset", offset))

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Read and send new lines
				newOffset, err := streamNewLines(file, conn, offset)
				if err != nil {
					log.Printf("Error streaming new lines: %v", err)
					return err
				}
				offset = newOffset
			}
		case err := <-watcher.Errors:
			log.Printf("Watcher error: %v", err)
			return err
		}
	}
}

// Helper function to read and send new lines from the file
func streamNewLines(file *os.File, conn *websocket.Conn, startOffset int64) (int64, error) {
	// Seek to the last offset
	_, err := file.Seek(startOffset, os.SEEK_SET)
	if err != nil {
		return startOffset, err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			return startOffset, err
		}
	}

	if err := scanner.Err(); err != nil {
		return startOffset, err
	}

	// Return new offset
	return file.Seek(0, io.SeekCurrent)
}

// Read last N lines of the file
func readLastLines(file *os.File, n int) ([]string, error) {
	var lines []string
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	size := stat.Size()
	buf := make([]byte, 1024)
	cursor := size
	var currentLine string

	for len(lines) < n && cursor > 0 {
		chunkSize := int64(len(buf))
		if cursor < chunkSize {
			chunkSize = cursor
		}

		cursor -= chunkSize
		file.Seek(cursor, os.SEEK_SET)
		readBytes, err := file.Read(buf[:chunkSize])
		if err != nil {
			return nil, err
		}

		currentLine = string(buf[:readBytes]) + currentLine
		parts := strings.Split(currentLine, "\n")
		if len(parts) > 1 {
			lines = append(parts[1:], lines...)
			currentLine = parts[0]
		}
	}
	if currentLine != "" && len(lines) < n {
		lines = append([]string{currentLine}, lines...)
	}

	// Return the last N lines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

// {year}{month}{day}.log
func taskLogFileName(date time.Time) string {
	year, month, day := date.UTC().Date()
	return fmt.Sprintf(
		"%d%02d%02d.log",
		year, month, day,
	)
}

// pb_data/logs/{taskLogFileName}.log
func taskLogFilePathDate(app *pocketbase.PocketBase, taskId string, dateTime time.Time) string {
	fileName := taskLogFileName(dateTime.UTC())
	return filepath.Join(
		app.DataDir(),
		LogsBasePath,
		taskId,
		fileName,
	)
}

// Helper function to get today's log file path
func taskTodayLogFilePath(app *pocketbase.PocketBase, taskId string) string {
    return taskLogFilePathDate(app, taskId, time.Now())
}

func createLogFile(app *pocketbase.PocketBase, taskId string) (*os.File, error) {
	filePath := taskTodayLogFilePath(app, taskId)
	logDir := filepath.Dir(filePath)
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		return nil, NewFailedCreateLogFileDirectoryError()
	}

	if _, err := os.Stat(filePath); err == nil {
		return os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	}
	return os.Create(filePath)
}
