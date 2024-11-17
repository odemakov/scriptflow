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
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

const (
    LogsBasePath = "logs"
)

func handleLogsWebSocket(app *pocketbase.PocketBase) echo.HandlerFunc {
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

        // Locate the log file
        logFilePath := taskLogFilePath(app, taskId)
        app.Logger().Debug("LogsWebSocket handler", slog.String("file", logFilePath))
        if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
            conn.WriteMessage(websocket.TextMessage, []byte("Log file not found"))
            return nil
        }

        if err := sendLastLines(conn, logFilePath, 10); err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "Failed to send last lines")
        }
        if err := watchFileChanges(conn, logFilePath, app); err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "Failed to watch file changes")
        }
        return nil
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
        if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
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
        line := scanner.Text()
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

// pb_data/logs/{year}{month}{day}.log
func taskLogFilePath(app *pocketbase.PocketBase, taskId string) string {
    created := time.Now()
    year, month, day := created.Date()
    fileName := fmt.Sprintf(
        "%d%02d%02d.log",
        year, month, day,
    )
    return filepath.Join(
        app.DataDir(),
        LogsBasePath,
        taskId,
        fileName,
    )
}

func createLogFile(app *pocketbase.PocketBase, taskId string) (*os.File, error) {
    filePath := taskLogFilePath(app, taskId)
    logDir := filepath.Dir(filePath)
    if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
        return nil, NewFailedCreateLogFileDirectoryError()
    }

    if _, err := os.Stat(filePath); err == nil {
        return os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
    }
    return os.Create(filePath)
}
