package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/mail"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/slack-go/slack"
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
		sf.app.Logger().Error("failed to retrieve subscriptions", slog.Any("error", err))
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
				sf.app.Logger().Error("failed to retrieve previous runs count", slog.Any("error", err))
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
		sf.app.Logger().Error("failed to create notification", slog.Any("error", err))
		return
	}

	// update subscription notified time
	_, err = sf.app.DB().Update(
		CollectionSubscriptions,
		dbx.Params{"notified": types.NowDateTime()},
		dbx.HashExp{"id": subscription.Id},
	).Execute()
	if err != nil {
		sf.app.Logger().Error("failed to update subscription", slog.Any("error", err))
	}
}

// Select {threshold} most recent runs newer than {subscription.notified}
// return count of runs with status in {subscription.events}
func retrieveConsecutiveRunsCount(db dbx.Builder, subscription SubscriptionItem) (int, error) {
	// SELECT id FROM runs
	// WHERE task='{taskId}' AND created > '{notified}'
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
			"active": true,
			"task":   run.Task,
		})

	// Execute the query and fetch the results
	var subscriptions []SubscriptionItem
	err := query.All(&subscriptions)
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// send notification
func (sf *ScriptFlow) sendNotification(notificationContext NotificationContext) error {
	mc := sf.buildMessageContext(notificationContext)

	channelType := notificationContext.Channel.GetString("type")
	if channelType == ChannelTypeEmail {
		message, err := sf.notificationEmailMessage(mc)
		if err != nil {
			return err
		}
		return sf.sendEmailNotification(mc.Subject, message, notificationContext.Channel)
	} else if channelType == ChannelTypeSlack {
		message, err := sf.notificationSlackMessage(mc)
		if err != nil {
			return err
		}
		return sf.sendSlackNotification(mc.Subject, message, notificationContext.Channel)
	} else {
		sf.app.Logger().Error("unknown channel type", slog.Any("type", channelType))
	}
	return nil
}

// send slack message
func (sf *ScriptFlow) sendSlackNotification(subject string, message string, channel *core.Record) error {
	config := NotificationSlackConfig{}
	err := channel.UnmarshalJSONField("config", &config)
	if err != nil {
		return err
	}
	api := slack.New(config.Token)
	_, _, err = api.PostMessage(
		config.Channel,
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
	)
	if err != nil {
		return err
	}

	return nil
}

// send email message
func (sf *ScriptFlow) sendEmailNotification(subject string, message string, channel *core.Record) error {
	config := NotificationEmailConfig{}
	err := channel.UnmarshalJSONField("config", &config)
	if err != nil {
		return err
	}
	mailerMessage := &mailer.Message{
		From: mail.Address{
			Address: sf.app.Settings().Meta.SenderAddress,
			Name:    sf.app.Settings().Meta.SenderName,
		},
		To:      []mail.Address{{Address: config.To}},
		Subject: subject,
		HTML:    message,
	}
	return sf.app.NewMailClient().Send(mailerMessage)
}

func (sf *ScriptFlow) buildMessageContext(nc NotificationContext) MessageContext {
	taskUrl := fmt.Sprintf(
		"%s/#/%s/%s/history",
		sf.app.Settings().Meta.AppURL,
		nc.Project.GetString("id"),
		nc.Task.GetString("id"),
	)
	runUrl := fmt.Sprintf(
		"%s/#/%s/%s/%s",
		sf.app.Settings().Meta.AppURL,
		nc.Project.GetString("id"),
		nc.Task.GetString("id"),
		nc.Run.GetString("id"),
	)

	return MessageContext{
		Header: sf.app.Settings().Meta.AppName,
		Subject: fmt.Sprintf(
			"[%s] <%s> %s",
			sf.app.Settings().Meta.AppName,
			nc.Subscription.GetString("name"),
			nc.Run.GetString("status"),
		),
		Item: MessageItem{
			Command:  nc.Run.GetString("command"),
			Host:     nc.Run.GetString("host"),
			Status:   nc.Run.GetString("status"),
			Error:    nc.Run.GetString("connection_error"),
			ExitCode: fmt.Sprintf("%d", nc.Run.GetInt("exit_code")),
			Created:  nc.Run.GetDateTime("created").String(),
			Updated:  nc.Run.GetDateTime("updated").String(),
		},
		TaskUrl:  taskUrl,
		TaskName: nc.Task.GetString("name"),
		RunUrl:   runUrl,
	}
}

//go:embed templates/*
var embeddedTemplates embed.FS

func (sf *ScriptFlow) notificationEmailMessage(mc MessageContext) (string, error) {
	// Parse the HTML template from the embedded file system
	tmpl, err := template.ParseFS(embeddedTemplates, "templates/notification_email_message.html")
	if err != nil {
		return "", err
	}
	// Execute the template and return as a string
	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, mc); err != nil {
		return "", err
	}
	return tpl.String(), nil
}

func (sf *ScriptFlow) notificationSlackMessage(mc MessageContext) (string, error) {
	// Parse the HTML template from the embedded file system
	tmpl, err := template.ParseFS(embeddedTemplates, "templates/notification_slack_message.md")
	if err != nil {
		return "", err
	}
	// Execute the template and return as a string
	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, mc); err != nil {
		return "", err
	}
	return tpl.String(), nil
}
