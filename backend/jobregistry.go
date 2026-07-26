package main

import (
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// JobRegistry encapsulates the bookkeeping between the database's active Tasks,
// the in-memory map of scheduled jobs, and the live gocron scheduler.
//
// It exposes a small surface (Schedule, Remove, Reconcile, IsScheduled) so that
// no other code needs to reach into activeJobs or the scheduler directly.
type JobRegistry struct {
	scheduler gocron.Scheduler
	runTask   func(taskId string)

	activeJobs map[string]gocron.Job
	jobsMutex  sync.RWMutex

	// scheduleMutex serializes concurrent Schedule calls so scheduler access
	// stays consistent, mirroring the previous ScriptFlowLocks.scheduleTask lock.
	scheduleMutex sync.Mutex
}

// NewJobRegistry creates a JobRegistry backed by the given scheduler.
// runTask is invoked (with the task id) whenever a scheduled job fires;
// a nil runTask is replaced with a no-op.
func NewJobRegistry(scheduler gocron.Scheduler, runTask func(taskId string)) *JobRegistry {
	if runTask == nil {
		runTask = func(string) {}
	}
	return &JobRegistry{
		scheduler:  scheduler,
		runTask:    runTask,
		activeJobs: make(map[string]gocron.Job),
	}
}

// Scheduler returns the underlying gocron scheduler, for callers that need to
// manage non-task (system) jobs or the scheduler's lifecycle (Start/Shutdown).
// Task-scoped scheduling state stays behind Schedule/Remove/Reconcile.
func (jr *JobRegistry) Scheduler() gocron.Scheduler {
	return jr.scheduler
}

// IsScheduled reports whether the given task currently has a job tracked
// in the registry. This is the only read surface exposed outside the module.
func (jr *JobRegistry) IsScheduled(taskId string) bool {
	jr.jobsMutex.RLock()
	defer jr.jobsMutex.RUnlock()
	_, exists := jr.activeJobs[taskId]
	return exists
}

// Schedule creates, updates, or removes the gocron job for the given task,
// keeping activeJobs in sync. Concurrent calls are serialized via scheduleMutex.
func (jr *JobRegistry) Schedule(app core.App, task *core.Record) error {
	jr.scheduleMutex.Lock()
	defer jr.scheduleMutex.Unlock()

	taskId := task.GetString("id")
	existingJob, jobExists := jr.getActiveJob(taskId)

	if !task.GetBool("active") {
		return jr.unschedule(app, task, existingJob, jobExists)
	}

	jobDefinition, err := jr.buildJobDefinition(app, task, taskId)
	if err != nil {
		return err
	}

	taskFunc := gocron.NewTask(jr.runTask, taskId)
	jobOptions := []gocron.JobOption{
		gocron.WithTags(taskId),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	}

	if jobExists {
		updatedJob, err := jr.scheduler.Update(existingJob.ID(), jobDefinition, taskFunc, jobOptions...)
		if err != nil {
			app.Logger().Error("failed to update task", taskAttrs(task), slog.Any("error", err))
			return err
		}
		jr.setActiveJob(taskId, updatedJob)
		app.Logger().Info("updated existing task", taskAttrs(task))
		return nil
	}

	newJob, err := jr.scheduler.NewJob(jobDefinition, taskFunc, jobOptions...)
	if err != nil {
		app.Logger().Error("failed to schedule new task", taskAttrs(task), slog.Any("error", err))
		return err
	}
	jr.setActiveJob(taskId, newJob)
	app.Logger().Info("scheduled new task", taskAttrs(task))
	return nil
}

// unschedule removes the job for an inactive task, if one exists.
func (jr *JobRegistry) unschedule(app core.App, task *core.Record, existingJob gocron.Job, jobExists bool) error {
	if !jobExists {
		return nil
	}
	if err := jr.scheduler.RemoveJob(existingJob.ID()); err != nil {
		app.Logger().Error("failed to remove inactive task", taskAttrs(task), slog.Any("error", err))
		return err
	}
	jr.removeActiveJob(task.GetString("id"))
	app.Logger().Info("removed inactive task", taskAttrs(task))
	return nil
}

// buildJobDefinition parses the task's schedule field into a gocron.JobDefinition,
// supporting "@every" durations, standard cron, and Jenkins-style H-notation.
func (jr *JobRegistry) buildJobDefinition(app core.App, task *core.Record, taskId string) (gocron.JobDefinition, error) {
	app.Logger().Info("schedule task", taskAttrs(task))
	schedule := task.GetString("schedule")

	if strings.HasPrefix(schedule, "@every ") {
		duration, err := time.ParseDuration(schedule[7:])
		if err != nil {
			app.Logger().Error("failed to parse duration", taskAttrs(task), slog.Any("error", err))
			return nil, err
		}
		// spread tasks by 10% of duration to avoid running them simultaneously
		min, max := durationMinMax(duration)
		return gocron.DurationRandomJob(min, max), nil
	}

	// Resolve Jenkins-style H notation if present
	resolvedSchedule, err := resolveHashedSchedule(schedule, taskId)
	if err != nil {
		app.Logger().Error("invalid hashed schedule", taskAttrs(task), slog.Any("error", err))
		return nil, err
	}
	if resolvedSchedule != schedule {
		app.Logger().Debug("resolved hashed schedule",
			slog.String("taskId", taskId),
			slog.String("original", schedule),
			slog.String("resolved", resolvedSchedule))
	}
	return gocron.CronJob(resolvedSchedule, false), nil
}

// Remove removes the job for taskId from both the scheduler and activeJobs.
// It is a safe no-op if the task has no scheduled job.
func (jr *JobRegistry) Remove(taskId string) error {
	job, exists := jr.getActiveJob(taskId)
	if !exists {
		return nil
	}
	if err := jr.scheduler.RemoveJob(job.ID()); err != nil {
		return err
	}
	jr.removeActiveJob(taskId)
	return nil
}

// Reconcile performs the two-stage synchronization: database -> activeJobs -> scheduler.
func (jr *JobRegistry) Reconcile(app core.App) {
	jr.reconcileActiveJobs(app)
	jr.reconcileScheduler(app)
}

// reconcileActiveJobs syncs database to activeJobs map (Stage 1)
// Ensures activeJobs map matches database: removes orphaned entries, adds missing entries
func (jr *JobRegistry) reconcileActiveJobs(app core.App) {
	activeTasks, err := jr.getActiveTasks(app)
	if err != nil {
		app.Logger().Error("failed to find active tasks during reconciliation", slog.Any("error", err))
		return
	}

	orphanedJobs, missingTasks := jr.findActiveJobsMismatches(activeTasks)

	orphanedCount := jr.removeOrphanedActiveJobs(app, orphanedJobs)
	scheduledCount := jr.scheduleMissingActiveTasks(app, missingTasks)

	if orphanedCount > 0 || scheduledCount > 0 {
		app.Logger().Info("activeJobs reconciliation completed",
			slog.Int("orphanedJobsRemoved", orphanedCount),
			slog.Int("missingTasksScheduled", scheduledCount))
	}
}

// getActiveTasks retrieves all active tasks from database
func (jr *JobRegistry) getActiveTasks(app core.App) ([]*core.Record, error) {
	return app.FindAllRecords(
		CollectionTasks,
		dbx.HashExp{"active": true},
	)
}

// findActiveJobsMismatches compares database tasks with activeJobs map
func (jr *JobRegistry) findActiveJobsMismatches(activeTasks []*core.Record) ([]string, []*core.Record) {
	// Create set of active task IDs from database
	activeTaskIds := make(map[string]bool)
	for _, task := range activeTasks {
		activeTaskIds[task.Id] = true
	}

	jr.jobsMutex.RLock()
	defer jr.jobsMutex.RUnlock()

	// Find orphaned jobs (in activeJobs but not in database)
	orphanedJobs := make([]string, 0)
	for taskId := range jr.activeJobs {
		if !activeTaskIds[taskId] {
			orphanedJobs = append(orphanedJobs, taskId)
		}
	}

	// Find missing tasks (in database but not in activeJobs)
	missingTasks := make([]*core.Record, 0)
	for _, task := range activeTasks {
		if _, exists := jr.activeJobs[task.Id]; !exists {
			missingTasks = append(missingTasks, task)
		}
	}

	return orphanedJobs, missingTasks
}

// removeOrphanedActiveJobs removes jobs from activeJobs map and scheduler
func (jr *JobRegistry) removeOrphanedActiveJobs(app core.App, orphanedJobs []string) int {
	count := 0
	jr.jobsMutex.Lock()
	defer jr.jobsMutex.Unlock()

	for _, taskId := range orphanedJobs {
		if job, exists := jr.activeJobs[taskId]; exists {
			if err := jr.scheduler.RemoveJob(job.ID()); err != nil {
				app.Logger().Error("failed to remove orphaned job from activeJobs",
					slog.String("taskId", taskId),
					slog.Any("error", err))
			} else {
				delete(jr.activeJobs, taskId)
				count++
				app.Logger().Info("removed orphaned job from activeJobs", slog.String("taskId", taskId))
			}
		}
	}
	return count
}

// scheduleMissingActiveTasks schedules missing tasks that exist in database but not in activeJobs
func (jr *JobRegistry) scheduleMissingActiveTasks(app core.App, missingTasks []*core.Record) int {
	count := 0
	for _, task := range missingTasks {
		app.Logger().Info("scheduling missing task from database", slog.String("taskId", task.Id))
		if err := jr.Schedule(app, task); err != nil {
			app.Logger().Error("failed to schedule missing task", slog.String("taskId", task.Id), slog.Any("error", err))
		}
		count++
	}
	return count
}

// reconcileScheduler syncs activeJobs map to gocron scheduler (Stage 2)
// Removes orphaned scheduler jobs and reschedules missing ones
func (jr *JobRegistry) reconcileScheduler(app core.App) {
	userJobs := jr.getUserJobs()
	orphanedJobs, missingTasks := jr.findSchedulerMismatches(userJobs)

	orphanedCount := jr.removeOrphanedJobs(app, orphanedJobs)
	rescheduledCount := jr.rescheduleMissingTasks(app, missingTasks)

	if orphanedCount > 0 || rescheduledCount > 0 {
		app.Logger().Info("scheduler reconciliation completed",
			slog.Int("orphanedSchedulerJobs", orphanedCount),
			slog.Int("rescheduledJobs", rescheduledCount))
	}
}

// getUserJobs returns all non-system jobs from scheduler
func (jr *JobRegistry) getUserJobs() []gocron.Job {
	allJobs := jr.scheduler.Jobs()
	userJobs := make([]gocron.Job, 0, len(allJobs))

	for _, job := range allJobs {
		if !slices.Contains(job.Tags(), SystemTask) {
			userJobs = append(userJobs, job)
		}
	}
	return userJobs
}

// findSchedulerMismatches compares scheduler jobs with activeJobs map
func (jr *JobRegistry) findSchedulerMismatches(userJobs []gocron.Job) ([]gocron.Job, []string) {
	jr.jobsMutex.RLock()
	defer jr.jobsMutex.RUnlock()

	// Find orphaned scheduler jobs (in scheduler but not in activeJobs map)
	orphanedJobs := make([]gocron.Job, 0)
	for _, job := range userJobs {
		if tags := job.Tags(); len(tags) > 0 {
			taskId := tags[0]
			if _, exists := jr.activeJobs[taskId]; !exists {
				orphanedJobs = append(orphanedJobs, job)
			}
		}
	}

	// Find missing tasks (in activeJobs map but not in scheduler)
	missingTasks := make([]string, 0)
	for taskId, mapJob := range jr.activeJobs {
		found := false
		for _, schedulerJob := range userJobs {
			if mapJob.ID() == schedulerJob.ID() {
				found = true
				break
			}
		}
		if !found {
			missingTasks = append(missingTasks, taskId)
		}
	}

	return orphanedJobs, missingTasks
}

// removeOrphanedJobs removes jobs from scheduler that don't exist in activeJobs map
func (jr *JobRegistry) removeOrphanedJobs(app core.App, orphanedJobs []gocron.Job) int {
	count := 0
	for _, job := range orphanedJobs {
		if err := jr.scheduler.RemoveJob(job.ID()); err != nil {
			app.Logger().Error("failed to remove orphaned job from scheduler",
				slog.String("jobId", job.ID().String()),
				slog.Any("error", err))
		} else {
			count++
			app.Logger().Info("removed orphaned job from scheduler",
				slog.String("jobId", job.ID().String()))
		}
	}
	return count
}

// rescheduleMissingTasks reschedules tasks that exist in activeJobs map but not in scheduler
func (jr *JobRegistry) rescheduleMissingTasks(app core.App, missingTasks []string) int {
	count := 0
	for _, taskId := range missingTasks {
		task, err := app.FindRecordById(CollectionTasks, taskId)
		if err != nil {
			app.Logger().Error("failed to find task for rescheduling",
				slog.String("taskId", taskId),
				slog.Any("error", err))
			// Remove stale map entry
			jr.removeActiveJob(taskId)
			continue
		}

		app.Logger().Info("rescheduling missing job", slog.String("taskId", taskId))
		if err := jr.Schedule(app, task); err != nil {
			app.Logger().Error("failed to reschedule missing task", slog.String("taskId", task.Id), slog.Any("error", err))
		}
		count++
	}
	return count
}

// Job map helper methods

func (jr *JobRegistry) getActiveJob(taskId string) (gocron.Job, bool) {
	jr.jobsMutex.RLock()
	defer jr.jobsMutex.RUnlock()
	job, exists := jr.activeJobs[taskId]
	return job, exists
}

func (jr *JobRegistry) setActiveJob(taskId string, job gocron.Job) {
	jr.jobsMutex.Lock()
	defer jr.jobsMutex.Unlock()
	jr.activeJobs[taskId] = job
}

func (jr *JobRegistry) removeActiveJob(taskId string) {
	jr.jobsMutex.Lock()
	defer jr.jobsMutex.Unlock()
	delete(jr.activeJobs, taskId)
}
