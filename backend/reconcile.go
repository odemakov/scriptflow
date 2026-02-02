package main

import (
	"log/slog"
	"slices"

	"github.com/go-co-op/gocron/v2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// reconcileJobs performs two-stage synchronization: database → activeJobs → scheduler
func (sf *ScriptFlow) reconcileJobs() {
	sf.reconcileActiveJobs()
	sf.reconcileScheduler()
}

// reconcileActiveJobs syncs database to activeJobs map (Stage 1)
// Ensures activeJobs map matches database: removes orphaned entries, adds missing entries
func (sf *ScriptFlow) reconcileActiveJobs() {
	activeTasks, err := sf.getActiveTasks()
	if err != nil {
		sf.app.Logger().Error("failed to find active tasks during reconciliation", slog.Any("error", err))
		return
	}

	orphanedJobs, missingTasks := sf.findActiveJobsMismatches(activeTasks)

	orphanedCount := sf.removeOrphanedActiveJobs(orphanedJobs)
	scheduledCount := sf.scheduleMissingActiveTasks(missingTasks)

	if orphanedCount > 0 || scheduledCount > 0 {
		sf.app.Logger().Info("activeJobs reconciliation completed",
			slog.Int("orphanedJobsRemoved", orphanedCount),
			slog.Int("missingTasksScheduled", scheduledCount))
	}
}

// getActiveTasks retrieves all active tasks from database
func (sf *ScriptFlow) getActiveTasks() ([]*core.Record, error) {
	return sf.app.FindAllRecords(
		CollectionTasks,
		dbx.HashExp{"active": true},
	)
}

// findActiveJobsMismatches compares database tasks with activeJobs map
func (sf *ScriptFlow) findActiveJobsMismatches(activeTasks []*core.Record) ([]string, []*core.Record) {
	// Create set of active task IDs from database
	activeTaskIds := make(map[string]bool)
	for _, task := range activeTasks {
		activeTaskIds[task.Id] = true
	}

	sf.jobsMutex.RLock()
	defer sf.jobsMutex.RUnlock()

	// Find orphaned jobs (in activeJobs but not in database)
	orphanedJobs := make([]string, 0)
	for taskId := range sf.activeJobs {
		if !activeTaskIds[taskId] {
			orphanedJobs = append(orphanedJobs, taskId)
		}
	}

	// Find missing tasks (in database but not in activeJobs)
	missingTasks := make([]*core.Record, 0)
	for _, task := range activeTasks {
		if _, exists := sf.activeJobs[task.Id]; !exists {
			missingTasks = append(missingTasks, task)
		}
	}

	return orphanedJobs, missingTasks
}

// removeOrphanedActiveJobs removes jobs from activeJobs map and scheduler
func (sf *ScriptFlow) removeOrphanedActiveJobs(orphanedJobs []string) int {
	count := 0
	sf.jobsMutex.Lock()
	defer sf.jobsMutex.Unlock()

	for _, taskId := range orphanedJobs {
		if job, exists := sf.activeJobs[taskId]; exists {
			if err := sf.scheduler.RemoveJob(job.ID()); err != nil {
				sf.app.Logger().Error("failed to remove orphaned job from activeJobs",
					slog.String("taskId", taskId),
					slog.Any("error", err))
			} else {
				delete(sf.activeJobs, taskId)
				count++
				sf.app.Logger().Info("removed orphaned job from activeJobs", slog.String("taskId", taskId))
			}
		}
	}
	return count
}

// scheduleMissingActiveTasks schedules missing tasks that exist in database but not in activeJobs
func (sf *ScriptFlow) scheduleMissingActiveTasks(missingTasks []*core.Record) int {
	count := 0
	for _, task := range missingTasks {
		sf.app.Logger().Info("scheduling missing task from database", slog.String("taskId", task.Id))
		go sf.ScheduleTask(task)
		count++
	}
	return count
}

// reconcileScheduler syncs activeJobs map to gocron scheduler (Stage 2)
// Removes orphaned scheduler jobs and reschedules missing ones
func (sf *ScriptFlow) reconcileScheduler() {
	userJobs := sf.getUserJobs()
	orphanedJobs, missingTasks := sf.findSchedulerMismatches(userJobs)

	orphanedCount := sf.removeOrphanedJobs(orphanedJobs)
	rescheduledCount := sf.rescheduleMissingTasks(missingTasks)

	if orphanedCount > 0 || rescheduledCount > 0 {
		sf.app.Logger().Info("scheduler reconciliation completed",
			slog.Int("orphanedSchedulerJobs", orphanedCount),
			slog.Int("rescheduledJobs", rescheduledCount))
	}
}

// getUserJobs returns all non-system jobs from scheduler
func (sf *ScriptFlow) getUserJobs() []gocron.Job {
	allJobs := sf.scheduler.Jobs()
	userJobs := make([]gocron.Job, 0, len(allJobs))

	for _, job := range allJobs {
		if !slices.Contains(job.Tags(), SystemTask) {
			userJobs = append(userJobs, job)
		}
	}
	return userJobs
}

// findSchedulerMismatches compares scheduler jobs with activeJobs map
func (sf *ScriptFlow) findSchedulerMismatches(userJobs []gocron.Job) ([]gocron.Job, []string) {
	sf.jobsMutex.RLock()
	defer sf.jobsMutex.RUnlock()

	// Find orphaned scheduler jobs (in scheduler but not in activeJobs map)
	orphanedJobs := make([]gocron.Job, 0)
	for _, job := range userJobs {
		if tags := job.Tags(); len(tags) > 0 {
			taskId := tags[0]
			if _, exists := sf.activeJobs[taskId]; !exists {
				orphanedJobs = append(orphanedJobs, job)
			}
		}
	}

	// Find missing tasks (in activeJobs map but not in scheduler)
	missingTasks := make([]string, 0)
	for taskId, mapJob := range sf.activeJobs {
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
func (sf *ScriptFlow) removeOrphanedJobs(orphanedJobs []gocron.Job) int {
	count := 0
	for _, job := range orphanedJobs {
		if err := sf.scheduler.RemoveJob(job.ID()); err != nil {
			sf.app.Logger().Error("failed to remove orphaned job from scheduler",
				slog.String("jobId", job.ID().String()),
				slog.Any("error", err))
		} else {
			count++
			sf.app.Logger().Info("removed orphaned job from scheduler",
				slog.String("jobId", job.ID().String()))
		}
	}
	return count
}

// rescheduleMissingTasks reschedules tasks that exist in activeJobs map but not in scheduler
func (sf *ScriptFlow) rescheduleMissingTasks(missingTasks []string) int {
	count := 0
	for _, taskId := range missingTasks {
		task, err := sf.app.FindRecordById(CollectionTasks, taskId)
		if err != nil {
			sf.app.Logger().Error("failed to find task for rescheduling",
				slog.String("taskId", taskId),
				slog.Any("error", err))
			// Remove stale map entry
			sf.jobsMutex.Lock()
			delete(sf.activeJobs, taskId)
			sf.jobsMutex.Unlock()
			continue
		}

		sf.app.Logger().Info("rescheduling missing job", slog.String("taskId", taskId))
		go sf.ScheduleTask(task)
		count++
	}
	return count
}
