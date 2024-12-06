package main

import (
	"log/slog"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// on run create/update checks notification configs and creates notification row if needed
func (sf *ScriptFlow) ProcessRunNotification(run *core.Record) {
	runItem := &RunItem{
		Id:     run.GetString("id"),
		Task:   run.GetString("task"),
		Status: run.GetString("status"),
	}
	subscriptions, err := retrieveSubscriptionsForRun(sf.app.DB(), runItem)
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
			sf.createNotification(&subscription, run)
		} else {
			consecutiveRunsCount, err := retrieveConsecutiveRunsCount(sf.app.DB(), subscription)
			if err != nil {
				sf.app.Logger().Error("failed to retrieve previous runs count", slog.Any("err", err))
				continue
			}

			if consecutiveRunsCount >= subscription.Threshold {
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

// Select {threshold} most recent runs with status in ({subscription.events}) for the task, newer than {subscriptio.notified}
// returns row's count
func retrieveConsecutiveRunsCount(db dbx.Builder, subscription SubscriptionItem) (int, error) {
	// SELECT id FROM runs
	// WHERE task='{taskId}' AND created > '{notified}' AND status IN ('status1', 'status2')
	// ORDER BY `created` DESC
	// LIMIT {threshold}
	query := db.Select("status").
		From(CollectionRuns).
		Where(dbx.And(
			dbx.HashExp{"task": subscription.Task},
			dbx.NewExp("created > {:created}", dbx.Params{"created": subscription.Notified}),
		)).
		OrderBy("created DESC").
		Limit(int64(subscription.Threshold))

	runs := []RunItem{}
	err := query.All(&runs)
	if err != nil {
		return 0, err
	}

	// create map of events to optimize search in the loop
	eventSet := make(map[string]struct{}, len(subscription.Events))
	for _, event := range subscription.Events {
		eventSet[event] = struct{}{}
	}

	cnt := 0
	for _, run := range runs {
		if _, exists := eventSet[run.Status]; exists {
			cnt++
		}
	}
	return cnt, nil
}

// retrieve subscriptions for the run
// consider only active subscriptions and those that have event matching the run status
func retrieveSubscriptionsForRun(db dbx.Builder, run *RunItem) ([]SubscriptionItem, error) {
	// SELECT DISTINCT subscriptions.*
	// FROM subscriptions
	// JOIN json_each(subscriptions.events) AS je ON je.value = 'error'
	// WHERE task = '{task}';
	query := db.Select("subscriptions.*").
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
