package main

import (
	"log/slog"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// on run create/update checks notification configs and creates notification row if needed
func (sf *ScriptFlow) ProcessRunNotification(run *core.Record) {
	// retrieve task
	task, err := retrieveTaskById(sf.app, run.GetString("task"))
	if err != nil {
		sf.app.Logger().Error("failed to retrieve task", slog.Any("err", err))
		return
	}

	runItem := &RunItem{
		Id:     run.GetString("id"),
		Task:   run.GetString("task"),
		Status: run.GetString("status"),
	}
	subscriptions, err := retrieveSubscriptionsForRun(sf.app, runItem)
	if err != nil {
		sf.app.Logger().Error("failed to retrieve subscriptions", slog.Any("err", err))
		return
	}

	// create notifications if needed
	for _, subscription := range subscriptions {
		if !subscription.Active {
			continue
		}

		sf.app.Logger().Debug("process subscription", slog.Any("subscription", subscription))
		if subscription.Threshold < 2 {
			// create notification
			sf.createNotification(&subscription, run)
		} else {
			// SELECT count(*) FROM runs WHERE task='taskId' AND created > 'created' AND status IN ('status1', 'status2') ORDER BY `created` DESC LIMIT {threshold};
			consecutiveRunsCount, err := retrieveConsecutiveRunsCount(sf.app, task, subscription)
			if err != nil {
				sf.app.Logger().Error("failed to retrieve previous runs count", slog.Any("err", err))
				continue
			}

			if consecutiveRunsCount >= subscription.Threshold {
				// create notification
				sf.createNotification(&subscription, run)
			}
		}
	}
}

func (sf *ScriptFlow) createNotification(subscription *SubscriptionItem, run *core.Record) {
	sf.app.Logger().Debug("create notification", slog.Any("subscription", subscription))

	// create notification
	_, err := sf.app.DB().Insert(
		CollectionNotifications,
		dbx.Params{
			"subscription": subscription.Id,
			"run":          run.Id,
			"created":      types.NowDateTime(),
			"updated":      types.NowDateTime(),
		},
	).Execute()
	if err != nil {
		sf.app.Logger().Error("failed to create notification", slog.Any("err", err))
		return
	}

	// update subscription notified time
	_, err = sf.app.DB().Update(
		CollectionSubscriptions,
		dbx.Params{"notified": types.NowDateTime()},
		dbx.HashExp{"id": subscription.Id},
	).Execute()
	if err != nil {
		sf.app.Logger().Error("failed to update subscription", slog.Any("err", err))
	}
}

func convertToAnyArray(jsonArray types.JSONArray[string]) []any {
	result := make([]any, len(jsonArray))
	for i, v := range jsonArray {
		result[i] = any(v)
	}
	return result
}

func retrieveConsecutiveRunsCount(app *pocketbase.PocketBase, task TaskItem, subscription SubscriptionItem) (int, error) {
	query := app.DB().
		Select("count(*) AS count").
		From(CollectionRuns).
		Where(dbx.And(
			dbx.HashExp{"task": task.Id},
			dbx.NewExp("created > {:created}", dbx.Params{"created": subscription.Notified}),
			dbx.In("status", convertToAnyArray(subscription.Events)...),
		)).
		OrderBy("created DESC").
		Limit(int64(subscription.Threshold))

	var count struct {
		Value int `db:"count"`
	}
	err := query.One(&count)
	if err != nil {
		return 0, err
	}
	return count.Value, nil
}

func retrieveTaskById(app *pocketbase.PocketBase, taskId string) (TaskItem, error) {
	query := app.DB().
		Select("*").
		From(CollectionTasks).
		Where(dbx.HashExp{"id": taskId})

	var task TaskItem
	err := query.One(&task)
	if err != nil {
		return TaskItem{}, err
	}
	return task, nil
}

// retrieve subscriptions for the run
// consider only active subscriptions and those that have event matching the run status
func retrieveSubscriptionsForRun(app *pocketbase.PocketBase, run *RunItem) ([]SubscriptionItem, error) {
	// SELECT DISTINCT subscriptions.*
	// FROM subscriptions
	// JOIN json_each(subscriptions.events) AS je ON je.value = 'error'
	// WHERE task = '{task}';
	query := app.DB().
		Select("subscriptions.*").
		From(CollectionSubscriptions).
		Join("JOIN", "json_each(subscriptions.events) AS je", dbx.HashExp{"je.value": run.Status}).
		Where(dbx.HashExp{
			"task":   run.Task,
			"active": true,
		})

	// Execute the query and fetch the results
	var subscriptions []SubscriptionItem
	err := query.All(&subscriptions)
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}
