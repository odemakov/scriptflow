package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

func TestResolveHashedSchedule(t *testing.T) {
	tests := []struct {
		name          string
		schedule      string
		seed          string
		expectError   bool
		errorContains string
		validate      func(t *testing.T, result string)
	}{
		{
			name:     "no H - unchanged",
			schedule: "10 0 * * *",
			seed:     "task1",
			validate: func(t *testing.T, result string) {
				assert.Equal(t, "10 0 * * *", result)
			},
		},
		{
			name:     "H in minute field",
			schedule: "H 0 * * *",
			seed:     "task1",
			validate: func(t *testing.T, result string) {
				// Should resolve to "N 0 * * *" where N is 0-59
				var minute int
				_, err := fmt.Sscanf(result, "%d 0 * * *", &minute)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, minute, 0)
				assert.LessOrEqual(t, minute, 59)
			},
		},
		{
			name:     "H(10-30) in minute field",
			schedule: "H(10-30) 0 * * *",
			seed:     "task1",
			validate: func(t *testing.T, result string) {
				var minute int
				_, err := fmt.Sscanf(result, "%d 0 * * *", &minute)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, minute, 10)
				assert.LessOrEqual(t, minute, 30)
			},
		},
		{
			name:     "H in hour field",
			schedule: "0 H * * *",
			seed:     "task1",
			validate: func(t *testing.T, result string) {
				var hour int
				_, err := fmt.Sscanf(result, "0 %d * * *", &hour)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, hour, 0)
				assert.LessOrEqual(t, hour, 23)
			},
		},
		{
			name:     "H(0-6) in hour field",
			schedule: "0 H(0-6) * * *",
			seed:     "task1",
			validate: func(t *testing.T, result string) {
				var hour int
				_, err := fmt.Sscanf(result, "0 %d * * *", &hour)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, hour, 0)
				assert.LessOrEqual(t, hour, 6)
			},
		},
		{
			name:     "multiple H fields",
			schedule: "H H * * *",
			seed:     "task1",
			validate: func(t *testing.T, result string) {
				var minute, hour int
				_, err := fmt.Sscanf(result, "%d %d * * *", &minute, &hour)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, minute, 0)
				assert.LessOrEqual(t, minute, 59)
				assert.GreaterOrEqual(t, hour, 0)
				assert.LessOrEqual(t, hour, 23)
			},
		},
		{
			name:     "deterministic - same seed same result",
			schedule: "H 0 * * *",
			seed:     "consistent-task",
			validate: func(t *testing.T, result string) {
				// Run again with same seed, should get same result
				result2, _ := resolveHashedSchedule("H 0 * * *", "consistent-task")
				assert.Equal(t, result, result2)
			},
		},
		{
			name:     "different seeds produce distribution",
			schedule: "H 0 * * *",
			seed:     "task-0",
			validate: func(t *testing.T, result string) {
				// Verify that different seeds produce at least 2 distinct values
				seen := make(map[string]bool)
				seen[result] = true
				for i := 1; i <= 10; i++ {
					r, _ := resolveHashedSchedule("H 0 * * *", fmt.Sprintf("task-%d", i))
					seen[r] = true
				}
				assert.GreaterOrEqual(t, len(seen), 2, "expected different seeds to produce distribution")
			},
		},
		{
			name:     "non-5-field cron unchanged",
			schedule: "*/5 * * * * *",
			seed:     "task1",
			validate: func(t *testing.T, result string) {
				assert.Equal(t, "*/5 * * * * *", result)
			},
		},
		{
			name:     "H(0-0) single value range",
			schedule: "H(0-0) 0 * * *",
			seed:     "task1",
			validate: func(t *testing.T, result string) {
				// Range of 1 (0-0) should always return 0
				assert.Equal(t, "0 0 * * *", result)
			},
		},
		// Error cases - Jenkins-style validation
		{
			name:          "error: min > max in minute field",
			schedule:      "H(30-10) 0 * * *",
			seed:          "task1",
			expectError:   true,
			errorContains: "min (30) > max (10)",
		},
		{
			name:          "error: min > max in hour field (wraparound)",
			schedule:      "0 H(22-6) * * *",
			seed:          "task1",
			expectError:   true,
			errorContains: "min (22) > max (6)",
		},
		{
			name:          "error: minute > 59",
			schedule:      "H(50-70) 0 * * *",
			seed:          "task1",
			expectError:   true,
			errorContains: "range out of bounds in minute field",
		},
		{
			name:          "error: hour > 23",
			schedule:      "0 H(20-30) * * *",
			seed:          "task1",
			expectError:   true,
			errorContains: "range out of bounds in hour field",
		},
		{
			name:          "error: day of month > 31",
			schedule:      "0 0 H(1-32) * *",
			seed:          "task1",
			expectError:   true,
			errorContains: "range out of bounds in day of month field",
		},
		{
			name:          "error: month > 12",
			schedule:      "0 0 1 H(1-13) *",
			seed:          "task1",
			expectError:   true,
			errorContains: "range out of bounds in month field",
		},
		{
			name:          "error: day of week > 6",
			schedule:      "0 0 * * H(0-7)",
			seed:          "task1",
			expectError:   true,
			errorContains: "range out of bounds in day of week field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveHashedSchedule(tt.schedule, tt.seed)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}
			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Mock job for testing
type mockJob struct {
	id   uuid.UUID
	tags []string
}

func (m *mockJob) ID() uuid.UUID                     { return m.id }
func (m *mockJob) Tags() []string                    { return m.tags }
func (m *mockJob) LastRun() (time.Time, error)       { return time.Time{}, nil }
func (m *mockJob) Name() string                      { return "" }
func (m *mockJob) NextRun() (time.Time, error)       { return time.Time{}, nil }
func (m *mockJob) NextRuns(int) ([]time.Time, error) { return nil, nil }
func (m *mockJob) RunNow() error                     { return nil }

// Mock scheduler for testing
type mockScheduler struct {
	jobs        []gocron.Job
	removedJobs []uuid.UUID
}

func (m *mockScheduler) Jobs() []gocron.Job {
	return m.jobs
}

func (m *mockScheduler) RemoveJob(id uuid.UUID) error {
	m.removedJobs = append(m.removedJobs, id)
	// Remove job from jobs slice
	for i, job := range m.jobs {
		if job.ID() == id {
			m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockScheduler) NewJob(gocron.JobDefinition, gocron.Task, ...gocron.JobOption) (gocron.Job, error) {
	return nil, nil
}
func (m *mockScheduler) Update(uuid.UUID, gocron.JobDefinition, gocron.Task, ...gocron.JobOption) (gocron.Job, error) {
	return nil, nil
}
func (m *mockScheduler) Start()                  {}
func (m *mockScheduler) StopJobs() error         { return nil }
func (m *mockScheduler) Shutdown() error         { return nil }
func (m *mockScheduler) JobsWaitingInQueue() int { return 0 }
func (m *mockScheduler) RemoveByTags(...string)  {}

func TestFormatLogLine(t *testing.T) {
	ts := time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		stream string
		out    string
		want   string
	}{
		{
			name:   "basic stdout line",
			stream: "stdout", out: "hello world",
			want: "[2025-06-10T12:00:00Z] [stdout] hello world\n",
		},
		{
			name:   "stderr line",
			stream: "stderr", out: "error occurred",
			want: "[2025-06-10T12:00:00Z] [stderr] error occurred\n",
		},
		{
			name:   "trailing newline stripped",
			stream: "stdout", out: "line\n",
			want: "[2025-06-10T12:00:00Z] [stdout] line\n",
		},
		{
			name:   "trailing CR+LF stripped",
			stream: "stdout", out: "line\r\n",
			want: "[2025-06-10T12:00:00Z] [stdout] line\n",
		},
		{
			name:   "empty output",
			stream: "stdout", out: "",
			want: "[2025-06-10T12:00:00Z] [stdout] \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLogLine(ts, tt.stream, tt.out)
			assert.Equal(t, tt.want, got)
			assert.True(t, strings.HasSuffix(got, "\n"), "must end with newline")
		})
	}
}
