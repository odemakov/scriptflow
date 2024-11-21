package main

import (
	"os"
	"testing"
)

func TestExtractLogsForRun(t *testing.T) {
	tests := []struct {
		name         string
		logContent   string
		runId        string
		expectedLogs []string
	}{
		{
			name: "Common case - single runId with logs",
			logContent: `[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug
Log line 1
Log line 2
[2024-11-21T18:00:00+01:00] [scriptflow] run another_run_id
Other log line`,
			runId: "85egyv91mcmw0ug",
			expectedLogs: []string{
				"[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug",
				"Log line 1",
				"Log line 2",
			},
		},
		{
			name: "Edge case - no delimiter at all",
			logContent: `Log line 1
Log line 2
Log line 3`,
			runId:        "85egyv91mcmw0ug",
			expectedLogs: []string{},
		},
		{
			name: "Edge case - no log lines between delimiters",
			logContent: `[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug
[2024-11-21T18:00:00+01:00] [scriptflow] run another_run_id`,
			runId: "85egyv91mcmw0ug",
			expectedLogs: []string{
				"[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug",
			},
		},
		{
			// theoretically, only one tash is write to the log file as the given time
			// all the tasks are singletones
			name: "Common case - multiple log lines for runId",
			logContent: `[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug
Log line 1
Log line 2
[2024-11-21T18:00:00+01:00] [scriptflow] run another_run_id
Other log line
[2024-11-21T18:01:00+01:00] [scriptflow] run 85egyv91mcmw0ug
Log line 3`,
			runId: "85egyv91mcmw0ug",
			expectedLogs: []string{
				"[2024-11-21T17:59:26+01:00] [scriptflow] run 85egyv91mcmw0ug",
				"Log line 1",
				"Log line 2",
			},
		},
		{
			name: "Edge case - runId not found",
			logContent: `[2024-11-21T17:59:26+01:00] [scriptflow] run another_run_id
Other log line`,
			runId:        "85egyv91mcmw0ug",
			expectedLogs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary log file
			tmpFile, err := os.CreateTemp("", "logfile-*.log")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write log content to the temporary file
			_, err = tmpFile.WriteString(tt.logContent)
			if err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Call the function
			logs, err := extractLogsForRun(tmpFile.Name(), tt.runId)
			if err != nil {
				t.Fatalf("Function returned an error: %v", err)
			}

			// Compare the result with the expected output
			if len(logs) != len(tt.expectedLogs) {
				t.Fatalf("Expected %d logs, got %d", len(tt.expectedLogs), len(logs))
			}
			for i, logLine := range logs {
				if logLine != tt.expectedLogs[i] {
					t.Errorf("Mismatch at line %d: expected %q, got %q", i, tt.expectedLogs[i], logLine)
				}
			}
		})
	}
}

/*
const LogsBasePath = "logs"

// Test taskLogFileName
func TestTaskLogFileName(t *testing.T) {
	tests := []struct {
		name       string
		inputDate  time.Time
		expected   string
	}{
		{
			name:      "Standard date",
			inputDate: time.Date(2024, 11, 21, 15, 0, 0, 0, time.UTC),
			expected:  "20241121.log",
		},
		{
			name:      "Leap year date",
			inputDate: time.Date(2024, 2, 29, 15, 0, 0, 0, time.UTC),
			expected:  "20240229.log",
		},
		{
			name:      "End of year date",
			inputDate: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			expected:  "20241231.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := taskLogFileName(tt.inputDate)
			if output != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, output)
			}
		})
	}
}

// Test taskLogFilePathDate
func TestTaskLogFilePathDate(t *testing.T) {
	tests := []struct {
		name        string
		inputDate   time.Time
		taskId      string
		expectedEnd string
	}{
		{
			name:        "Standard date with taskId",
			inputDate:   time.Date(2024, 11, 21, 15, 0, 0, 0, time.UTC),
			taskId:      "exampleTask",
			expectedEnd: filepath.Join("logs", "exampleTask", "20241121.log"),
		},
		{
			name:        "Leap year date with taskId",
			inputDate:   time.Date(2024, 2, 29, 15, 0, 0, 0, time.UTC),
			taskId:      "leapTask",
			expectedEnd: filepath.Join("logs", "leapTask", "20240229.log"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockApp := createMockPocketBaseApp(t)
			output := taskLogFilePathDate(mockApp, tt.taskId, tt.inputDate)
			expectedPath := filepath.Join(mockApp.DataDir(), tt.expectedEnd)

			if output != expectedPath {
				t.Errorf("Expected %s, got %s", expectedPath, output)
			}
		})
	}
}

// Test taskTodayLogFilePath
func TestTaskTodayLogFilePath(t *testing.T) {
	mockApp := createMockPocketBaseApp(t)
	taskId := "exampleTask"
	today := time.Now().UTC()
	expectedEnd := filepath.Join("logs", taskId, taskLogFileName(today))

	t.Run("Today's log file path", func(t *testing.T) {
		output := taskTodayLogFilePath(mockApp, taskId)
		expectedPath := filepath.Join(mockApp.DataDir(), expectedEnd)

		if output != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, output)
		}
	})
}

// Helper to create a mock PocketBase app
func createMockPocketBaseApp(t *testing.T) *pocketbase.PocketBase {
	tempDir, err := os.MkdirTemp("", "pb_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	mockApp := &pocketbase.PocketBase{}
	mockApp.SetDataDir(tempDir)

	return mockApp
}
*/