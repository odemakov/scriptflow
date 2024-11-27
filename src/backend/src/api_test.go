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