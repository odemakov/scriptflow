package main

import (
	"testing"
	"time"
)

func TestTaskFileDate(t *testing.T) {
	sf := &ScriptFlow{}

	tests := []struct {
		fileName    string
		expected    time.Time
		expectError bool
		errorType   error
	}{
		// Valid case
		{"20231201.log", time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC), false, nil},
		// Invalid file name format
		{"invalid_name.txt", time.Time{}, true, NewInvalidLogFileNameError()},
		// Incorrect date format
		{"20231301.log", time.Time{}, true, NewFailedParseDateFromLogFileNameError()},
		// File name too short
		{"2023.log", time.Time{}, true, NewInvalidLogFileNameError()},
		// File name too long
		{"2023010110.log", time.Time{}, true, NewInvalidLogFileNameError()},
	}

	for _, test := range tests {
		t.Run(test.fileName, func(t *testing.T) {
			result, err := sf.taskFileDate(test.fileName)
			if (err != nil) != test.expectError {
				t.Errorf("expected error: %v, got: %v", test.expectError, err)
			}

			if err != nil && test.errorType != nil && err.Error() != test.errorType.Error() {
				t.Errorf("expected error type: %v, got: %v", test.errorType, err)
			}

			if !result.Equal(test.expected) {
				t.Errorf("expected date: %v, got: %v", test.expected, result)
			}
		})
	}
}
