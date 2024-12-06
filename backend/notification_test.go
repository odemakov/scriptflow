package main

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/stretchr/testify/assert"
)

func TestRetrieveConsecutiveRunsCount(t *testing.T) {
	testApp, _ := tests.NewTestApp()
	defer testApp.Cleanup()

	CreateRunsCollection()

	type runStatusAndDate struct {
		status  string
		created types.DateTime
	}
	tests := []struct {
		name          string
		runs          []runStatusAndDate
		expectedCount int
	}{
		{
			name:          "no runs",
			runs:          []runStatusAndDate{},
			expectedCount: 0,
		},
		{
			name: "One completed run",
			runs: []runStatusAndDate{
				{status: "completed", created: types.NowDateTime().Add(-time.Hour)},
			},
			expectedCount: 0,
		},
		{
			name: "Two completed run",
			runs: []runStatusAndDate{
				{status: "completed", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-2 * time.Hour)},
			},
			expectedCount: 0,
		},
		{
			name: "Three completed run",
			runs: []runStatusAndDate{
				{status: "completed", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-2 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-3 * time.Hour)},
			},
			expectedCount: 0,
		},
		{
			name: "One error and three completed run",
			runs: []runStatusAndDate{
				{status: "error", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-2 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-3 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-4 * time.Hour)},
			},
			expectedCount: 1,
		},
		{
			name: "Two error and two completed run",
			runs: []runStatusAndDate{
				{status: "error", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "error", created: types.NowDateTime().Add(-2 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-3 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-4 * time.Hour)},
			},
			expectedCount: 2,
		},
		{
			name: "Three error and one completed run",
			runs: []runStatusAndDate{
				{status: "error", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "error", created: types.NowDateTime().Add(-2 * time.Hour)},
				{status: "error", created: types.NowDateTime().Add(-3 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-5 * time.Hour)},
			},
			expectedCount: 3,
		},
		{
			name: "Four_error_runs",
			runs: []runStatusAndDate{
				{status: "error", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "error", created: types.NowDateTime().Add(-2 * time.Hour)},
				{status: "error", created: types.NowDateTime().Add(-3 * time.Hour)},
				{status: "error", created: types.NowDateTime().Add(-5 * time.Hour)},
			},
			expectedCount: 3, // we limit result to subscription.Threshold
		},
		{
			name: "Two_error_one_completed_one_error_runs",
			runs: []runStatusAndDate{
				{status: "error", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "error", created: types.NowDateTime().Add(-2 * time.Hour)},
				{status: "completed", created: types.NowDateTime().Add(-3 * time.Hour)},
				{status: "error", created: types.NowDateTime().Add(-5 * time.Hour)},
			},
			expectedCount: 2,
		},
		{
			name: "Tree_error_two_of_them_before_notified",
			runs: []runStatusAndDate{
				{status: "error", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "error", created: types.NowDateTime().AddDate(0, 0, -2)},
				{status: "error", created: types.NowDateTime().AddDate(0, 0, -2)},
			},
			expectedCount: 1,
		},
		{
			name: "Tree_error_two_of_them_before_notified",
			runs: []runStatusAndDate{
				{status: "error", created: types.NowDateTime().Add(-1 * time.Hour)},
				{status: "error", created: types.NowDateTime().AddDate(0, 0, -2)},
				{status: "error", created: types.NowDateTime().AddDate(0, 0, -2)},
				{status: "completed", created: types.NowDateTime().AddDate(0, 0, -2)},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := SubscriptionItem{
				Task:      core.GenerateDefaultRandomId(),
				Threshold: 3,
				Events:    types.JSONArray[string]{"error"},
				Notified:  types.NowDateTime().AddDate(0, 0, -1),
			}
			// Insert runs
			for _, run := range tt.runs {
				testApp.DB().Insert(CollectionRuns, dbx.Params{
					"task":    subscription.Task,
					"status":  run.status,
					"created": run.created,
				}).Execute()
			}

			count, err := retrieveConsecutiveRunsCount(testApp.DB(), subscription)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, count)

			testApp.DB().Delete(CollectionRuns, dbx.HashExp{"task": subscription.Task}).Execute()
		})
	}
}

func TestRetrieveSubscriptionsForRun(t *testing.T) {
	testApp, _ := tests.NewTestApp()
	defer testApp.Cleanup()

	CreateSubscriptionsCollection()

	tests := []struct {
		name           string
		subscriptions  []SubscriptionItem
		run            RunItem
		expectedResult int
	}{
		{
			name:           "no subscriptions",
			subscriptions:  []SubscriptionItem{},
			run:            RunItem{Task: "task1", Status: "error"},
			expectedResult: 0,
		},
		{
			name: "one matching subscription",
			subscriptions: []SubscriptionItem{
				{
					Id:       "sub1",
					Task:     "task1",
					Active:   true,
					Events:   types.JSONArray[string]{"error"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
			},
			run:            RunItem{Task: "task1", Status: "error"},
			expectedResult: 1,
		},
		{
			name: "one non-matching subscription",
			subscriptions: []SubscriptionItem{
				{
					Id:       "sub1",
					Task:     "task1",
					Active:   true,
					Events:   types.JSONArray[string]{"completed"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
			},
			run:            RunItem{Task: "task1", Status: "error"},
			expectedResult: 0,
		},
		{
			name: "multiple matching subscriptions",
			subscriptions: []SubscriptionItem{
				{
					Id:       "sub1",
					Task:     "task1",
					Active:   true,
					Events:   types.JSONArray[string]{"error"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
				{
					Id:       "sub2",
					Task:     "task1",
					Active:   true,
					Events:   types.JSONArray[string]{"error"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
			},
			run:            RunItem{Task: "task1", Status: "error"},
			expectedResult: 2,
		},
		{
			name: "multiple matching subscriptions with different events",
			subscriptions: []SubscriptionItem{
				{
					Id:       "sub1",
					Task:     "task1",
					Active:   true,
					Events:   types.JSONArray[string]{"error", "error_connection"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
				{
					Id:       "sub2",
					Task:     "task1",
					Active:   true,
					Events:   types.JSONArray[string]{"error", "complete"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
			},
			run:            RunItem{Task: "task1", Status: "error"},
			expectedResult: 2,
		},
		{
			name: "multiple non matching subscriptions with different events",
			subscriptions: []SubscriptionItem{
				{
					Id:       "sub1",
					Task:     "task1",
					Active:   true,
					Events:   types.JSONArray[string]{"error", "error_connection"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
				{
					Id:       "sub2",
					Task:     "task1",
					Active:   true,
					Events:   types.JSONArray[string]{"error", "complete"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
			},
			run:            RunItem{Task: "task1", Status: "interrupted"},
			expectedResult: 0,
		},
		{
			name: "inactive subscription",
			subscriptions: []SubscriptionItem{
				{
					Id:       "sub1",
					Task:     "task1",
					Active:   false,
					Events:   types.JSONArray[string]{"error"},
					Notified: types.NowDateTime().AddDate(0, 0, -1),
				},
			},
			run:            RunItem{Task: "task1", Status: "error"},
			expectedResult: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Insert subscriptions
			for _, subscription := range tt.subscriptions {
				testApp.DB().Insert(CollectionSubscriptions, dbx.Params{
					"id":       subscription.Id,
					"task":     subscription.Task,
					"active":   subscription.Active,
					"events":   subscription.Events,
					"notified": subscription.Notified,
				}).Execute()
			}

			subscriptions, err := retrieveSubscriptionsForRun(testApp.DB(), &tt.run)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, len(subscriptions))

			testApp.DB().Delete(CollectionSubscriptions, dbx.HashExp{"task": tt.run.Task}).Execute()
		})
	}
}

func CreateSubscriptionsCollection() *core.Collection {
	collection := core.NewBaseCollection(CollectionSubscriptions)
	collection.Fields.Add(&core.TextField{Name: "task"})
	collection.Fields.Add(&core.BoolField{Name: "active"})
	collection.Fields.Add(&core.JSONField{Name: "events"})
	collection.Fields.Add(&core.DateField{Name: "notified"})
	return collection
}

func CreateRunsCollection() *core.Collection {
	collection := core.NewBaseCollection(CollectionTasks)
	collection.Fields.Add(&core.TextField{Name: "name"})
	collection.Fields.Add(&core.TextField{Name: "command"})
	collection.Fields.Add(&core.TextField{Name: "schedule"})
	collection.Fields.Add(&core.TextField{Name: "node"})
	collection.Fields.Add(&core.TextField{Name: "project"})
	collection.Fields.Add(&core.BoolField{Name: "active"})
	collection.Fields.Add(&core.BoolField{Name: "prepend_datetime"})
	return collection
}
