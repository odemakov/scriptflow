package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Projects      []ConfigProject       `yaml:"projects"`
	Nodes         []ConfigNode          `yaml:"nodes"`
	Tasks         []ConfigTask          `yaml:"tasks"`
	Channels      []ConfigChannel       `yaml:"channels"`
	Subscriptions []ConfigSubscriptions `yaml:"subscriptions"`
}

type ConfigProject struct {
	Id     string              `yaml:"id"`
	Name   string              `yaml:"name"`
	Config ConfigProjectConfig `yaml:"config"`
}

type ConfigProjectConfig struct {
	LogsMaxDays int `yaml:"logs_max_days" json:"logsMaxDays,omitempty"`
}

type ConfigNode struct {
	Id         string `yaml:"id"`
	Host       string `yaml:"host"`
	Username   string `yaml:"username"`
	PrivateKey string `yaml:"private_key"`
}

type ConfigTask struct {
	Id              string `yaml:"id"`
	Name            string `yaml:"name"`
	Command         string `yaml:"command"`
	Schedule        string `yaml:"schedule"`
	Node            string `yaml:"node"`
	Project         string `yaml:"project"`
	Active          bool   `yaml:"active"`
	PrependDatetime bool   `yaml:"prepend_datetime"`
}

type ConfigChannel struct {
	Id     string              `yaml:"id"`
	Name   string              `yaml:"name"`
	Type   string              `yaml:"type"`
	Config ConfigChannelConfig `yaml:"config"`
}

type ConfigChannelConfig struct {
	To      string `yaml:"to" json:"to,omitempty"`
	Token   string `yaml:"token" json:"token,omitempty"`
	Channel string `yaml:"channel" json:"channel,omitempty"`
}

type ConfigSubscriptions struct {
	Id        string   `yaml:"id"`
	Name      string   `yaml:"name"`
	Task      string   `yaml:"task"`
	Channel   string   `yaml:"channel"`
	Events    []string `yaml:"events"`
	Threshold int      `yaml:"threshold"`
	Active    bool     `yaml:"active"`
}

func NewConfig(configFile string) (*Config, error) {
	// read yml file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	// Unmarshal the YAML data into the Config struct
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// return config
	return &config, nil
}

// function which inserts or updates database records according to the config file
func (sf *ScriptFlow) UpdateFromConfig() error {
	sf.configMutex.RLock()
	defer sf.configMutex.RUnlock()

	sf.updateFromConfigProject()
	sf.updateFromConfigNode()
	sf.updateFromConfigTasks()
	sf.updateFromConfigChannels()
	sf.updateFromConfigSubscriptions()
	return nil
}

func (sf *ScriptFlow) updateFromConfigProject() {
	// insert or update projects
	for _, project := range sf.config.Projects {
		// skip empty name
		if project.Name == "" {
			sf.app.Logger().Warn("[config] project name is empty", slog.Any("project", project))
			continue
		}
		if project.Id == "" {
			project.Id = generateIdFromName(project.Name)
		}
		if !isValidUUID(project.Id) {
			sf.app.Logger().Warn("[config] project id is not a valid UUID", slog.Any("project", project))
			continue
		}
		// format config as JSON string
		configJSON, err := json.Marshal(project.Config)
		if err != nil {
			sf.app.Logger().Error("[config] failed to marshal project config to JSON", slog.Any("error", err))
			continue
		}
		err = sf.insertOrUpdate(CollectionProjects, dbx.Params{
			"id":     project.Id,
			"name":   project.Name,
			"config": string(configJSON),
		}, "name", "config")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update project", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow) updateFromConfigNode() {
	// insert or update nodes
	for _, node := range sf.config.Nodes {
		// skip empty host, username
		if node.Host == "" || node.Username == "" {
			sf.app.Logger().Warn("[config] node id, host or username is empty", slog.Any("node", node))
			continue
		}
		if node.Id == "" {
			// generate id from name-host
			node.Id = generateIdFromName(node.Host + "-" + node.Username)
		}
		if !isValidUUID(node.Id) {
			sf.app.Logger().Warn("[config] node id is not a valid UUID", slog.Any("node", node))
			continue
		}
		err := sf.insertOrUpdate(CollectionNodes, dbx.Params{
			"id":          node.Id,
			"host":        node.Host,
			"username":    node.Username,
			"private_key": node.PrivateKey,
		}, "host", "username", "private_key")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update node", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow) updateFromConfigTasks() {
	// insert or update tasks
	for _, task := range sf.config.Tasks {
		// skip empty name, command, schedule, node, project
		if task.Name == "" || task.Command == "" || task.Schedule == "" || task.Node == "" || task.Project == "" {
			sf.app.Logger().Warn("[config] task id, name, command, schedule, node or project is empty", slog.Any("task", task))
			continue
		}
		if task.Id == "" {
			task.Id = generateIdFromName(task.Name)
		}
		if !isValidUUID(task.Id) {
			sf.app.Logger().Warn("[config] task id is not a valid UUID", slog.Any("task", task))
			continue
		}
		err := sf.insertOrUpdate(CollectionTasks, dbx.Params{
			"id":               task.Id,
			"name":             task.Name,
			"command":          task.Command,
			"schedule":         task.Schedule,
			"node":             task.Node,
			"project":          task.Project,
			"active":           task.Active,
			"prepend_datetime": task.PrependDatetime,
		}, "name", "command", "schedule", "node", "project", "active", "prepend_datetime")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update task", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow) updateFromConfigChannels() {
	// insert or update channels
	for _, channel := range sf.config.Channels {
		// skip empty name or type
		if channel.Name == "" || channel.Type == "" {
			sf.app.Logger().Warn("[config] channel id, name or type is empty", slog.Any("channel", channel))
			continue
		}
		if channel.Id == "" {
			channel.Id = generateIdFromName(channel.Name)
		}
		if !isValidUUID(channel.Id) {
			sf.app.Logger().Warn("[config] channel id is not a valid UUID", slog.Any("channel", channel))
			continue
		}
		if channel.Type != ChannelTypeSlack && channel.Type != ChannelTypeEmail {
			sf.app.Logger().Warn("[config] channel type is not supported", slog.Any("channel", channel))
			continue
		}
		// format config as JSON string
		configJSON, err := json.Marshal(channel.Config)
		if err != nil {
			sf.app.Logger().Error("[config] failed to marshal channel config to JSON", slog.Any("error", err))
			continue
		}
		err = sf.insertOrUpdate(CollectionChannels, dbx.Params{
			"id":     channel.Id,
			"name":   channel.Name,
			"type":   channel.Type,
			"config": string(configJSON),
		}, "name", "type", "config")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update channel", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow) updateFromConfigSubscriptions() {
	events := []string{RunStatusStarted, RunStatusError, RunStatusCompleted, RunStatusInterrupted, RunStatusInternalError}

	// insert or update subscriptions
	for _, subscription := range sf.config.Subscriptions {
		// skip empty name, channel
		if subscription.Name == "" || subscription.Channel == "" || subscription.Task == "" {
			sf.app.Logger().Warn("[config] subscription id, name, channel or task is empty", slog.Any("subscription", subscription))
			continue
		}
		if subscription.Id == "" {
			subscription.Id = generateIdFromName(subscription.Name)
		}

		if subscription.Events == nil {
			sf.app.Logger().Warn("[config] subscription events is empty", slog.Any("subscription", subscription))
			continue
		}
		eventsList, err := sf.subscriptionFilterOut(&subscription.Events, &events)
		if err != nil {
			sf.app.Logger().Warn("[config] subscription events error", slog.Any("error", err), slog.Any("subscription", subscription))
			continue
		}

		err = sf.insertOrUpdate(CollectionSubscriptions, dbx.Params{
			"id":        subscription.Id,
			"name":      subscription.Name,
			"task":      subscription.Task,
			"channel":   subscription.Channel,
			"events":    string(eventsList),
			"threshold": subscription.Threshold,
			"active":    subscription.Active,
		}, "name", "task", "channel", "threshold", "active")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update subscription", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow) subscriptionFilterOut(configValues *[]string, correctValues *[]string) ([]byte, error) {
	var validValues []string
	for _, configValue := range *configValues {
		for _, correctValue := range *correctValues {
			if configValue == correctValue {
				validValues = append(validValues, configValue)
				break
			}
		}
	}
	// format result to JSON array
	res, err := json.Marshal(validValues)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (sf *ScriptFlow) insertOrUpdate(table string, params dbx.Params, updateColumns ...string) error {
	setClause := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		setClause[i] = fmt.Sprintf("%s={:%s}", col, col)
	}

	query := sf.app.DB().NewQuery(fmt.Sprintf(
		`INSERT INTO %s (%s,created,updated) VALUES (%s,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP) ON CONFLICT (id) DO UPDATE SET %s,updated=CURRENT_TIMESTAMP`,
		table,
		strings.Join(keys(params), ","),
		strings.Join(placeholders(params), ","),
		strings.Join(setClause, ","),
	))
	query.Bind(params)
	_, err := query.Execute()
	return err
}

func keys(params dbx.Params) []string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Ensure keys are sorted
	return keys
}

func placeholders(params dbx.Params) []string {
	keys := keys(params) // Get sorted keys
	placeholders := make([]string, len(keys))
	for i, k := range keys {
		placeholders[i] = "{:" + k + "}"
	}
	return placeholders
}

func isValidUUID(s string) bool {
	re := regexp.MustCompile(`^[a-z][a-z0-9-]{5,}$`)
	return re.MatchString(s)
}

// generate id from name
// to lower case
// replace all non-alphanumeric characters with '-'
// replace all multiple '-' with single '-'
// remove prefix digits
// trim '-' from start and end
func generateIdFromName(name string) string {
	id := strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	id = re.ReplaceAllString(id, "-")
	re = regexp.MustCompile(`-+`)
	id = re.ReplaceAllString(id, "-")
	re = regexp.MustCompile(`^[0-9]+`)
	id = re.ReplaceAllString(id, "")
	id = strings.Trim(id, "-")
	return id
}
