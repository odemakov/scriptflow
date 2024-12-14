package main

import (
	"fmt"
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
func TestDurationMinMax(t *testing.T) {
	tests := []struct {
		duration time.Duration
		min      time.Duration
		max      time.Duration
	}{
		{time.Second * 10, time.Second * 9, time.Second * 11},
		{time.Minute * 5, time.Second * 270, time.Second * 330},
		{time.Hour, time.Second * 3240, time.Second * 3960},
		{time.Millisecond * 100, time.Millisecond * 90, time.Millisecond * 110},
		{time.Second, time.Millisecond * 900, time.Millisecond * 1100},
		{time.Minute * 10, time.Minute * 9, time.Minute * 11},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("duration=%v", test.duration), func(t *testing.T) {
			min, max := durationMinMax(test.duration)
			if min != test.min {
				t.Errorf("expected min: %v, got: %v", test.min, min)
			}
			if max != test.max {
				t.Errorf("expected max: %v, got: %v", test.max, max)
			}
		})
	}
}
