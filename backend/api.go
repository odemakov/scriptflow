package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var (
	openWebSockets      int
	openWebSocketsMutex sync.Mutex
	// Pre-compiled regex for log delimiter matching
	logDelimiterRegex = regexp.MustCompile(`^\[.*\] \[scriptflow\] run (\S+)$`)
)

func (sf *ScriptFlow) authenticateWebSocketConnection(conn *websocket.Conn, taskId string) (*core.Record, error) {
	// Wait for authentication message with timeout
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	// Read the first message, which should contain authentication token
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		sf.app.Logger().Warn("WebSocket auth failed",
			slog.String("error", "Failed to read authentication message"),
			slog.String("taskId", taskId))
		return nil, err
	}

	// Reset deadline for future messages
	conn.SetReadDeadline(time.Time{})

	// Parse the authentication message
	var authMessage struct {
		Token string `json:"token"`
	}

	if messageType != websocket.TextMessage || json.Unmarshal(message, &authMessage) != nil ||
		authMessage.Token == "" {
		sf.app.Logger().Warn("WebSocket auth failed",
			slog.String("error", "Invalid authentication message format"),
			slog.String("taskId", taskId))
		conn.WriteMessage(websocket.TextMessage, []byte(
			`{"status":"error","message":"Authentication required"}`))
		return nil, fmt.Errorf("invalid auth message format")
	}

	// Validate the PocketBase token
	authRecord, err := sf.app.FindAuthRecordByToken(
		authMessage.Token,
		core.TokenTypeAuth,
	)

	if err != nil || authRecord == nil {
		sf.app.Logger().Warn("WebSocket auth failed",
			slog.String("error", fmt.Sprintf("%v", err)),
			slog.String("taskId", taskId))
		conn.WriteMessage(websocket.TextMessage, []byte(
			`{"status":"error","message":"Invalid authentication token"}`))
		return nil, fmt.Errorf("invalid auth token")
	}

	return authRecord, nil
}

func (sf *ScriptFlow) ApiTaskLogWebSocket(e *core.RequestEvent) error {
	taskId := e.Request.PathValue("taskId")

	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Adjust as needed for security
		},
	}

	conn, err := upgrader.Upgrade(e.Response, e.Request, nil)
	if err != nil {
		return e.InternalServerError(err.Error(), "Failed to upgrade WebSocket")
	}
	defer conn.Close()

	// Authenticate the connection
	_, err = sf.authenticateWebSocketConnection(conn, taskId)
	if err != nil {
		return e.Next() // End connection on authentication failure
	}

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
	logFilePath := sf.taskTodayLogFilePath(taskId)
	sf.app.Logger().Debug("TaskLogWebSocket handler", slog.String("file", logFilePath))
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		err := conn.WriteMessage(websocket.TextMessage, []byte("Log file not found"))
		if err != nil {
			return e.InternalServerError(err.Error(), "Failed to write to socket")
		}
		return e.Next()
	}

	if err := sendLastLines(conn, logFilePath, 100); err != nil {
		return e.InternalServerError(err.Error(), "Failed to send last lines")
	}
	if err := watchFileChanges(conn, logFilePath, sf.app); err != nil {
		return e.InternalServerError(err.Error(), "Failed to watch file changes")
	}
	return e.Next()
}

// function to retrieve run log by runId
func (sf *ScriptFlow) ApiRunLog(e *core.RequestEvent) error {
	runId := e.Request.PathValue("runId")

	// select run by run id
	run, err := sf.app.FindRecordById(CollectionRuns, runId)
	if err != nil {
		return e.NotFoundError("Run not found", slog.String("runId", runId))
	}

	// select task by run
	task, err := sf.app.FindRecordById(CollectionTasks, run.GetString("task"))
	if err != nil {
		return e.NotFoundError("Task not found", slog.String("taskId", run.GetString("task")))
	}

	// get log file path
	logFilePath := sf.taskLogFilePathDate(
		task.GetString("id"),
		run.GetDateTime("created").Time(),
	)
	logs, err := extractLogsForRun(logFilePath, runId)
	if err != nil {
		return e.InternalServerError(err.Error(), slog.String("runId", runId))
	}

	// return {data: logs: []string}
	return e.JSON(http.StatusOK, map[string]interface{}{
		"logs": logs,
	})
}

func (sf *ScriptFlow) ApiScriptFlowStats(e *core.RequestEvent) error {
	openWebSocketsMutex.Lock()
	count := openWebSockets
	openWebSocketsMutex.Unlock()
	return e.JSON(http.StatusOK, map[string]int{"WebSocketsCount": count})
}

func (sf *ScriptFlow) ApiKillRun(e *core.RequestEvent) error {
	runId := e.Request.PathValue("runId")

	if err := sf.KillRun(runId); err != nil {
		return e.NotFoundError(err.Error(), nil)
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "killed", "runId": runId})
}

// ApiLatestRuns returns the most recent run per task in a single query,
// replacing N individual SDK calls from the frontend task list view.
func (sf *ScriptFlow) ApiLatestRuns(e *core.RequestEvent) error {
	taskIdsParam := e.Request.URL.Query().Get("taskIds")
	if taskIdsParam == "" {
		return e.JSON(http.StatusOK, map[string]RunItem{})
	}

	taskIds := strings.Split(taskIdsParam, ",")
	if len(taskIds) == 0 {
		return e.JSON(http.StatusOK, map[string]RunItem{})
	}

	// Build placeholders for parameterized query
	params := dbx.Params{}
	placeholders := make([]string, len(taskIds))
	for i, id := range taskIds {
		key := fmt.Sprintf("t%d", i)
		params[key] = id
		placeholders[i] = "{:" + key + "}"
	}

	query := sf.app.DB().NewQuery(fmt.Sprintf(`
		SELECT r.* FROM runs r
		INNER JOIN (
			SELECT task, MAX(created) as max_created FROM runs
			WHERE task IN (%s)
			GROUP BY task
		) latest ON r.task = latest.task AND r.created = latest.max_created
	`, strings.Join(placeholders, ",")))
	query.Bind(params)

	var runs []RunItem
	if err := query.All(&runs); err != nil {
		return e.InternalServerError(err.Error(), nil)
	}

	result := make(map[string]RunItem, len(runs))
	for _, run := range runs {
		result[run.Task] = run
	}

	return e.JSON(http.StatusOK, result)
}

func extractLogsForRun(logFilePath, runId string) ([]string, error) {
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
		if matches := logDelimiterRegex.FindStringSubmatch(line); matches != nil {
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
	offset, err := file.Seek(0, io.SeekStart)
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
	_, err := file.Seek(startOffset, io.SeekStart)
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
		offset, err := file.Seek(cursor, io.SeekStart)
		if err != nil {
			return nil, err
		}

		if offset != cursor {
			return nil, io.ErrUnexpectedEOF
		}

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
func TaskLogFileName(date time.Time) string {
	year, month, day := date.UTC().Date()
	return fmt.Sprintf(
		"%d%02d%02d.log",
		year, month, day,
	)
}
