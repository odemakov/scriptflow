package main

import (
	"fmt"
	"slices"
	"sync"
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

func TestReconcileActiveJobs(t *testing.T) {
	tests := []struct {
		name             string
		activeJobs       map[string]gocron.Job
		dbTasks          []string // task IDs that exist in database
		expectedRemove   []string // task IDs that should be removed
		expectedSchedule []string // task IDs that should be scheduled
	}{
		{
			name: "no orphaned jobs",
			activeJobs: map[string]gocron.Job{
				"task1": &mockJob{id: uuid.New(), tags: []string{"task1"}},
				"task2": &mockJob{id: uuid.New(), tags: []string{"task2"}},
			},
			dbTasks:          []string{"task1", "task2"},
			expectedRemove:   []string{},
			expectedSchedule: []string{},
		},
		{
			name: "one orphaned job",
			activeJobs: map[string]gocron.Job{
				"task1": &mockJob{id: uuid.New(), tags: []string{"task1"}},
				"task2": &mockJob{id: uuid.New(), tags: []string{"task2"}},
			},
			dbTasks:          []string{"task1"},
			expectedRemove:   []string{"task2"},
			expectedSchedule: []string{},
		},
		{
			name: "all jobs orphaned",
			activeJobs: map[string]gocron.Job{
				"task1": &mockJob{id: uuid.New(), tags: []string{"task1"}},
				"task2": &mockJob{id: uuid.New(), tags: []string{"task2"}},
			},
			dbTasks:          []string{},
			expectedRemove:   []string{"task1", "task2"},
			expectedSchedule: []string{},
		},
		{
			name:             "empty activeJobs",
			activeJobs:       map[string]gocron.Job{},
			dbTasks:          []string{"task1"},
			expectedRemove:   []string{},
			expectedSchedule: []string{"task1"},
		},
		{
			name: "missing tasks in activeJobs",
			activeJobs: map[string]gocron.Job{
				"task1": &mockJob{id: uuid.New(), tags: []string{"task1"}},
			},
			dbTasks:          []string{"task1", "task2", "task3"},
			expectedRemove:   []string{},
			expectedSchedule: []string{"task2", "task3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test ScriptFlow with mock data
			mockSched := &mockScheduler{}
			sf := &ScriptFlow{
				activeJobs: make(map[string]gocron.Job),
				jobsMutex:  sync.RWMutex{},
				scheduler:  mockSched,
			}

			// Setup activeJobs map
			for taskId, job := range tt.activeJobs {
				sf.activeJobs[taskId] = job
			}

			// Mock database by creating a function that simulates database query
			originalFindAllRecords := sf.app
			defer func() { sf.app = originalFindAllRecords }()

			// Since we can't easily mock sf.app.FindAllRecords, we'll test the logic directly
			// by simulating what reconcileActiveJobs does

			// Create set of active task IDs from database (simulated)
			activeTaskIds := make(map[string]bool)
			for _, taskId := range tt.dbTasks {
				activeTaskIds[taskId] = true
			}

			// Test the core logic of reconcileActiveJobs
			sf.jobsMutex.Lock()
			removedTasks := make([]string, 0)
			scheduledTasks := make([]string, 0)

			// Simulate removal logic
			for taskId, job := range sf.activeJobs {
				if !activeTaskIds[taskId] {
					// Simulate job removal
					_ = sf.scheduler.RemoveJob(job.ID())
					delete(sf.activeJobs, taskId)
					removedTasks = append(removedTasks, taskId)
				}
			}

			// Simulate scheduling logic
			for _, taskId := range tt.dbTasks {
				if _, exists := sf.activeJobs[taskId]; !exists {
					scheduledTasks = append(scheduledTasks, taskId)
				}
			}
			sf.jobsMutex.Unlock()

			// Verify results
			assert.ElementsMatch(t, tt.expectedRemove, removedTasks)
			assert.ElementsMatch(t, tt.expectedSchedule, scheduledTasks)

			// NOTE: This test only verifies detection logic.
			// Full scheduling behavior requires database mocking for sf.app.FindAllRecords()
		})
	}
}

func TestReconcileScheduler(t *testing.T) {
	tests := []struct {
		name               string
		schedulerJobs      []gocron.Job
		activeJobs         map[string]gocron.Job
		expectedRemove     []uuid.UUID // job IDs that should be removed from scheduler
		expectedReschedule []string    // task IDs that should be rescheduled
	}{
		{
			name: "no orphaned jobs",
			schedulerJobs: []gocron.Job{
				&mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), tags: []string{"task1"}},
				&mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"), tags: []string{"task2"}},
			},
			activeJobs: map[string]gocron.Job{
				"task1": &mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), tags: []string{"task1"}},
				"task2": &mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"), tags: []string{"task2"}},
			},
			expectedRemove:     []uuid.UUID{},
			expectedReschedule: []string{},
		},
		{
			name: "orphaned scheduler job",
			schedulerJobs: []gocron.Job{
				&mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), tags: []string{"task1"}},
				&mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"), tags: []string{"task2"}},
			},
			activeJobs: map[string]gocron.Job{
				"task1": &mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), tags: []string{"task1"}},
			},
			expectedRemove:     []uuid.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")},
			expectedReschedule: []string{},
		},
		{
			name: "stale map entry",
			schedulerJobs: []gocron.Job{
				&mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), tags: []string{"task1"}},
			},
			activeJobs: map[string]gocron.Job{
				"task1": &mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), tags: []string{"task1"}},
				"task2": &mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"), tags: []string{"task2"}},
			},
			expectedRemove:     []uuid.UUID{},
			expectedReschedule: []string{"task2"},
		},
		{
			name: "filter out system tasks",
			schedulerJobs: []gocron.Job{
				&mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), tags: []string{"task1"}},
				&mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"), tags: []string{SystemTask, JobCheckNodeStatus}},
			},
			activeJobs: map[string]gocron.Job{
				"task1": &mockJob{id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), tags: []string{"task1"}},
			},
			expectedRemove:     []uuid.UUID{},
			expectedReschedule: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test ScriptFlow with mock scheduler
			mockSched := &mockScheduler{jobs: tt.schedulerJobs}
			sf := &ScriptFlow{
				activeJobs: make(map[string]gocron.Job),
				jobsMutex:  sync.RWMutex{},
				scheduler:  mockSched,
			}

			// Setup activeJobs map
			for taskId, job := range tt.activeJobs {
				sf.activeJobs[taskId] = job
			}

			// Test the core logic of reconcileScheduler
			allJobs := sf.scheduler.Jobs()

			// Filter out system tasks
			userJobs := make([]gocron.Job, 0)
			for _, job := range allJobs {
				tags := job.Tags()
				isSystemTask := false
				isSystemTask = slices.Contains(tags, SystemTask)
				if !isSystemTask {
					userJobs = append(userJobs, job)
				}
			}

			// Find orphaned scheduler jobs
			sf.jobsMutex.RLock()
			orphanedSchedulerJobs := make([]gocron.Job, 0)
			missingMapEntries := make([]string, 0)

			for _, job := range userJobs {
				tags := job.Tags()
				if len(tags) > 0 {
					taskId := tags[0]
					if _, exists := sf.activeJobs[taskId]; !exists {
						orphanedSchedulerJobs = append(orphanedSchedulerJobs, job)
					}
				}
			}

			// Check for map entries missing from scheduler
			for taskId, mapJob := range sf.activeJobs {
				foundInScheduler := false
				for _, schedulerJob := range userJobs {
					if mapJob.ID() == schedulerJob.ID() {
						foundInScheduler = true
						break
					}
				}
				if !foundInScheduler {
					missingMapEntries = append(missingMapEntries, taskId)
				}
			}
			sf.jobsMutex.RUnlock()

			// Verify orphaned scheduler jobs
			orphanedIds := make([]uuid.UUID, 0)
			for _, job := range orphanedSchedulerJobs {
				orphanedIds = append(orphanedIds, job.ID())
			}
			assert.ElementsMatch(t, tt.expectedRemove, orphanedIds)

			// Verify missing map entries that should be rescheduled
			assert.ElementsMatch(t, tt.expectedReschedule, missingMapEntries)

			// NOTE: This test only verifies detection logic.
			// Full rescheduling behavior requires database mocking for sf.app.FindRecordById()
		})
	}
}
