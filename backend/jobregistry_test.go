package main

import (
	"testing"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

// schedulingMockScheduler extends the existing mockScheduler so that NewJob/Update
// return real tracked jobs instead of (nil, nil). This lets JobRegistry.Schedule's
// create/update code paths (and Reconcile's calls into it) run to completion without
// producing untracked nil jobs, mirroring gocron's real addOrUpdate semantics for Update.
type schedulingMockScheduler struct {
	mockScheduler
}

func newSchedulingMockScheduler(jobs ...gocron.Job) *schedulingMockScheduler {
	return &schedulingMockScheduler{mockScheduler{jobs: jobs}}
}

func (s *schedulingMockScheduler) NewJob(gocron.JobDefinition, gocron.Task, ...gocron.JobOption) (gocron.Job, error) {
	job := &mockJob{id: uuid.New()}
	s.jobs = append(s.jobs, job)
	return job, nil
}

func (s *schedulingMockScheduler) Update(id uuid.UUID, _ gocron.JobDefinition, _ gocron.Task, _ ...gocron.JobOption) (gocron.Job, error) {
	job := &mockJob{id: id}
	for i, j := range s.jobs {
		if j.ID() == id {
			s.jobs[i] = job
			return job, nil
		}
	}
	s.jobs = append(s.jobs, job)
	return job, nil
}

// newJobRegistryTestApp returns a throwaway PocketBase test app, already carrying
// the Tasks collection (registered by the project's migrations, and re-asserted here
// via the existing CreateRunsCollection() fixture helper).
func newJobRegistryTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	CreateRunsCollection()

	return app
}

// createTestTask inserts a Task record directly into the test app's database,
// bypassing collection validation (node/project relations aren't needed here).
func createTestTask(t *testing.T, app core.App, id, schedule string, active bool) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(CollectionTasks)
	if err != nil {
		t.Fatal(err)
	}

	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", "task-"+id)
	record.Set("command", "echo test")
	record.Set("schedule", schedule)
	record.Set("active", active)
	if err := app.SaveNoValidate(record); err != nil {
		t.Fatal(err)
	}
	return record
}

// newTestTaskRecord builds an in-memory Task record (not persisted), for tests that
// call Schedule directly without needing the record to be readable back from the db.
func newTestTaskRecord(t *testing.T, app core.App, id, schedule string, active bool) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(CollectionTasks)
	if err != nil {
		t.Fatal(err)
	}

	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", "task-"+id)
	record.Set("command", "echo test")
	record.Set("schedule", schedule)
	record.Set("active", active)
	return record
}

func TestJobRegistrySchedule(t *testing.T) {
	app := newJobRegistryTestApp(t)

	t.Run("new active task with standard cron schedule", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)
		task := newTestTaskRecord(t, app, "task1", "0 0 * * *", true)

		err := jr.Schedule(app, task)

		assert.NoError(t, err)
		assert.True(t, jr.IsScheduled("task1"))
	})

	t.Run("new active task with @every schedule", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)
		task := newTestTaskRecord(t, app, "task1", "@every 5m", true)

		err := jr.Schedule(app, task)

		assert.NoError(t, err)
		assert.True(t, jr.IsScheduled("task1"))
	})

	t.Run("new active task with jenkins-style H schedule", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)
		task := newTestTaskRecord(t, app, "task1", "H 0 * * *", true)

		err := jr.Schedule(app, task)

		assert.NoError(t, err)
		assert.True(t, jr.IsScheduled("task1"))
	})

	t.Run("active task with invalid @every duration", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)
		task := newTestTaskRecord(t, app, "task1", "@every not-a-duration", true)

		err := jr.Schedule(app, task)

		assert.Error(t, err)
		assert.False(t, jr.IsScheduled("task1"))
	})

	t.Run("active task with invalid hashed schedule", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)
		task := newTestTaskRecord(t, app, "task1", "H(30-10) 0 * * *", true)

		err := jr.Schedule(app, task)

		assert.Error(t, err)
		assert.False(t, jr.IsScheduled("task1"))
	})

	t.Run("update existing active task reschedules in place", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)
		task := newTestTaskRecord(t, app, "task1", "0 0 * * *", true)

		assert.NoError(t, jr.Schedule(app, task))
		task.Set("schedule", "0 12 * * *")
		err := jr.Schedule(app, task)

		assert.NoError(t, err)
		assert.True(t, jr.IsScheduled("task1"))
		assert.Empty(t, sched.removedJobs)
	})

	t.Run("deactivating a task removes its existing job", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)
		task := newTestTaskRecord(t, app, "task1", "0 0 * * *", true)
		assert.NoError(t, jr.Schedule(app, task))

		task.Set("active", false)
		err := jr.Schedule(app, task)

		assert.NoError(t, err)
		assert.False(t, jr.IsScheduled("task1"))
		assert.Len(t, sched.removedJobs, 1)
	})

	t.Run("inactive task without an existing job is a no-op", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)
		task := newTestTaskRecord(t, app, "task1", "0 0 * * *", false)

		err := jr.Schedule(app, task)

		assert.NoError(t, err)
		assert.False(t, jr.IsScheduled("task1"))
		assert.Empty(t, sched.removedJobs)
	})
}

func TestJobRegistryRemove(t *testing.T) {
	t.Run("removes an existing job", func(t *testing.T) {
		job := &mockJob{id: uuid.New(), tags: []string{"task1"}}
		sched := newSchedulingMockScheduler(job)
		jr := NewJobRegistry(sched, nil)
		jr.activeJobs["task1"] = job

		err := jr.Remove("task1")

		assert.NoError(t, err)
		assert.False(t, jr.IsScheduled("task1"))
		assert.Equal(t, []uuid.UUID{job.id}, sched.removedJobs)
	})

	t.Run("is a safe no-op for a task with no scheduled job", func(t *testing.T) {
		sched := newSchedulingMockScheduler()
		jr := NewJobRegistry(sched, nil)

		err := jr.Remove("unknown-task")

		assert.NoError(t, err)
		assert.Empty(t, sched.removedJobs)
	})
}

// TestJobRegistryReconcileDbToMap covers the same scenarios as the old
// TestReconcileActiveJobs (database -> activeJobs stage), but drives them through
// the real JobRegistry.Reconcile end to end.
func TestJobRegistryReconcileDbToMap(t *testing.T) {
	tests := []struct {
		name             string
		dbTaskIds        []string // active tasks seeded in the database
		activeJobs       []string // task ids pre-populated in activeJobs (and mirrored in the scheduler)
		expectedRemoved  []string // task ids expected to be removed from the scheduler
		expectedSchedule []string // task ids expected to end up scheduled
	}{
		{
			name:             "no orphaned jobs",
			dbTaskIds:        []string{"task1", "task2"},
			activeJobs:       []string{"task1", "task2"},
			expectedRemoved:  []string{},
			expectedSchedule: []string{"task1", "task2"},
		},
		{
			name:             "one orphaned job",
			dbTaskIds:        []string{"task1"},
			activeJobs:       []string{"task1", "task2"},
			expectedRemoved:  []string{"task2"},
			expectedSchedule: []string{"task1"},
		},
		{
			name:             "all jobs orphaned",
			dbTaskIds:        []string{},
			activeJobs:       []string{"task1", "task2"},
			expectedRemoved:  []string{"task1", "task2"},
			expectedSchedule: []string{},
		},
		{
			name:             "empty activeJobs",
			dbTaskIds:        []string{"task1"},
			activeJobs:       []string{},
			expectedRemoved:  []string{},
			expectedSchedule: []string{"task1"},
		},
		{
			name:             "missing tasks in activeJobs",
			dbTaskIds:        []string{"task1", "task2", "task3"},
			activeJobs:       []string{"task1"},
			expectedRemoved:  []string{},
			expectedSchedule: []string{"task1", "task2", "task3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newJobRegistryTestApp(t)
			for _, id := range tt.dbTaskIds {
				createTestTask(t, app, id, "0 0 * * *", true)
			}

			sched := newSchedulingMockScheduler()
			jr := NewJobRegistry(sched, nil)
			jobsByTaskId := map[string]*mockJob{}
			for _, id := range tt.activeJobs {
				job := &mockJob{id: uuid.New(), tags: []string{id}}
				jobsByTaskId[id] = job
				jr.activeJobs[id] = job
				sched.jobs = append(sched.jobs, job)
			}

			jr.Reconcile(app)

			removedIds := make([]uuid.UUID, 0, len(tt.expectedRemoved))
			for _, id := range tt.expectedRemoved {
				removedIds = append(removedIds, jobsByTaskId[id].id)
			}
			assert.ElementsMatch(t, removedIds, sched.removedJobs)

			for _, id := range tt.expectedSchedule {
				assert.True(t, jr.IsScheduled(id), "expected %s to be scheduled", id)
			}
			for _, id := range tt.expectedRemoved {
				assert.False(t, jr.IsScheduled(id), "expected %s to not be scheduled", id)
			}
		})
	}
}

// TestJobRegistryReconcileMapToScheduler covers the same scenarios as the old
// TestReconcileScheduler (activeJobs -> scheduler stage), but drives them through
// the real JobRegistry.Reconcile end to end. The database always mirrors activeJobs
// exactly so the first (db -> map) stage is a no-op, isolating the second stage.
func TestJobRegistryReconcileMapToScheduler(t *testing.T) {
	t.Run("no orphaned jobs", func(t *testing.T) {
		app := newJobRegistryTestApp(t)
		createTestTask(t, app, "task1", "0 0 * * *", true)
		createTestTask(t, app, "task2", "0 0 * * *", true)

		job1 := &mockJob{id: uuid.New(), tags: []string{"task1"}}
		job2 := &mockJob{id: uuid.New(), tags: []string{"task2"}}
		sched := newSchedulingMockScheduler(job1, job2)
		jr := NewJobRegistry(sched, nil)
		jr.activeJobs["task1"] = job1
		jr.activeJobs["task2"] = job2

		jr.Reconcile(app)

		assert.Empty(t, sched.removedJobs)
		assert.True(t, jr.IsScheduled("task1"))
		assert.True(t, jr.IsScheduled("task2"))
	})

	t.Run("orphaned scheduler job", func(t *testing.T) {
		app := newJobRegistryTestApp(t)
		createTestTask(t, app, "task1", "0 0 * * *", true)

		job1 := &mockJob{id: uuid.New(), tags: []string{"task1"}}
		job2 := &mockJob{id: uuid.New(), tags: []string{"task2"}}
		sched := newSchedulingMockScheduler(job1, job2)
		jr := NewJobRegistry(sched, nil)
		jr.activeJobs["task1"] = job1

		jr.Reconcile(app)

		assert.Equal(t, []uuid.UUID{job2.id}, sched.removedJobs)
		assert.True(t, jr.IsScheduled("task1"))
		assert.False(t, jr.IsScheduled("task2"))
	})

	t.Run("stale map entry gets rescheduled", func(t *testing.T) {
		app := newJobRegistryTestApp(t)
		createTestTask(t, app, "task1", "0 0 * * *", true)
		createTestTask(t, app, "task2", "0 0 * * *", true)

		job1 := &mockJob{id: uuid.New(), tags: []string{"task1"}}
		job2 := &mockJob{id: uuid.New(), tags: []string{"task2"}} // exists in map, missing from scheduler
		sched := newSchedulingMockScheduler(job1)
		jr := NewJobRegistry(sched, nil)
		jr.activeJobs["task1"] = job1
		jr.activeJobs["task2"] = job2

		jr.Reconcile(app)

		assert.Empty(t, sched.removedJobs)
		assert.True(t, jr.IsScheduled("task1"))
		assert.True(t, jr.IsScheduled("task2"))
	})

	t.Run("stale map entry for a task deleted from the database", func(t *testing.T) {
		app := newJobRegistryTestApp(t)
		createTestTask(t, app, "task1", "0 0 * * *", true)
		// task2 has a map entry but no longer exists in the database at all.

		job1 := &mockJob{id: uuid.New(), tags: []string{"task1"}}
		job2 := &mockJob{id: uuid.New(), tags: []string{"task2"}}
		sched := newSchedulingMockScheduler(job1)
		jr := NewJobRegistry(sched, nil)
		jr.activeJobs["task1"] = job1
		jr.activeJobs["task2"] = job2

		jr.Reconcile(app)

		assert.True(t, jr.IsScheduled("task1"))
		assert.False(t, jr.IsScheduled("task2"), "stale map entry for a deleted task should be dropped")
	})

	t.Run("system tasks are filtered out of the comparison", func(t *testing.T) {
		app := newJobRegistryTestApp(t)
		createTestTask(t, app, "task1", "0 0 * * *", true)

		job1 := &mockJob{id: uuid.New(), tags: []string{"task1"}}
		sysJob := &mockJob{id: uuid.New(), tags: []string{SystemTask, JobCheckNodeStatus}}
		sched := newSchedulingMockScheduler(job1, sysJob)
		jr := NewJobRegistry(sched, nil)
		jr.activeJobs["task1"] = job1

		jr.Reconcile(app)

		assert.Empty(t, sched.removedJobs)
		assert.True(t, jr.IsScheduled("task1"))
	})
}
