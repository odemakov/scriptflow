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
	sf.updateFromConfigProject()
	sf.updateFromConfigNode()
	sf.updateFromConfigTaks()
	sf.updateFromConfigChannels()
	sf.updateFromConfigSubscriptions()
	return nil
}

func (sf *ScriptFlow) updateFromConfigProject() {
	// insert or update projects
	for _, project := range sf.config.Projects {
		// skip empty id or name
		if project.Id == "" || project.Name == "" {
			sf.app.Logger().Warn("[config] project id or name is empty", slog.Any("project", project))
			continue
		}
		if !isValidUUID(project.Id) {
			sf.app.Logger().Warn("[config] project id is not a valid UUID", slog.Any("project", project))
			continue
		}
		// format config as JSON string
		configJSON, err := json.MarshalIndent(project.Config, "", "  ")
		if err != nil {
			sf.app.Logger().Error("[config] failed to marshal project config to JSON", slog.Any("error", err))
			continue
		}
		err = sf.insertOrUpdate(CollectionProjects, dbx.Params{
			"id":     project.Id,
			"name":   project.Name,
			"config": string(configJSON),
		}, "id", "name", "config")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update project", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow) updateFromConfigNode() {
	// insert or update nodes
	for _, node := range sf.config.Nodes {
		// skip empty host, username
		if node.Id == "" || node.Host == "" || node.Username == "" {
			sf.app.Logger().Warn("[config] node id, host or username is empty", slog.Any("node", node))
			continue
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
		}, "id", "host", "username", "private_key")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update node", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow) updateFromConfigTaks() {
	// insert or update tasks
	for _, task := range sf.config.Tasks {
		// skip empty id, name, command, schedule, node, project
		if task.Id == "" || task.Name == "" || task.Command == "" || task.Schedule == "" || task.Node == "" || task.Project == "" {
			sf.app.Logger().Warn("[config] task id, name, command, schedule, node or project is empty", slog.Any("task", task))
			continue
		}
		if !isValidUUID(task.Id) {
			sf.app.Logger().Warn("[config] task id is not a valid UUID", slog.Any("node", task))
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
		}, "id", "name", "command", "schedule", "node", "project", "active", "prepend_datetime")
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
			sf.app.Logger().Warn("[config] channel name or type is empty", slog.Any("channel", channel))
			continue
		}
		if channel.Type != ChannelTypeSlack && channel.Type != ChannelTypeEmail {
			sf.app.Logger().Warn("[config] channel type is not supported", slog.Any("channel", channel))
			continue
		}
		// format config as JSON string
		configJSON, err := json.MarshalIndent(channel.Config, "", "  ")
		if err != nil {
			sf.app.Logger().Error("[config] failed to marshal channel config to JSON", slog.Any("error", err))
			continue
		}
		err = sf.insertOrUpdate(CollectionChannels, dbx.Params{
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
	tasks, err := sf.createMapFromQuery("SELECT id, slug FROM tasks", []string{"slug"})
	if err != nil {
		sf.app.Logger().Error("[config] failed to select tasks", slog.Any("error", err))
		return
	}

	channels, err := sf.createMapFromQuery("SELECT id, name FROM channels", []string{"name"})
	if err != nil {
		sf.app.Logger().Error("[config] failed to select channels", slog.Any("error", err))
		return
	}

	// insert or update subscriptions
	for _, subscription := range sf.config.Subscriptions {
		// skip empty name, task or channel
		if subscription.Name == "" || subscription.Task == "" || subscription.Channel == "" {
			sf.app.Logger().Warn("[config] subscription name, task or channel is empty", slog.Any("subscription", subscription))
			continue
		}
		/* check event is in
		   RunStatusStarted       = "started"
		   RunStatusError         = "error"
		   RunStatusCompleted     = "completed"
		   RunStatusInterrupted   = "interrupted"
		   RunStatusInternalError = "internal_error"
		*/
		var events []string
		if subscription.Events != nil {
			for _, event := range subscription.Events {
				if event != RunStatusStarted && event != RunStatusError && event != RunStatusCompleted && event != RunStatusInterrupted && event != RunStatusInternalError {
					sf.app.Logger().Warn("[config] subscription event is not supported", slog.Any("subscription", subscription))
				} else {
					events = append(events, event)
				}
			}
		}
		// format events to JSON array
		eventsJSON, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			sf.app.Logger().Error("[config] failed to marshal subscription events to JSON", slog.Any("error", err))
			continue
		}

		err = sf.insertOrUpdate(CollectionSubscriptions, dbx.Params{
			"name":      subscription.Name,
			"task":      tasks[subscription.Task],
			"channel":   channels[subscription.Channel],
			"events":    string(eventsJSON),
			"threshold": subscription.Threshold,
			"active":    subscription.Active,
		}, "name,task,channel", "threshold", "active")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update subscription", slog.Any("error", err))
		}
	}
}

// This function creates a map from the query result
// map value is the first column of the query
// map key is the concatenation of the rest of the columns
//
// For query SELECT id, host, username FROM nodes
// returns map[node.host+","+node.username] = node.id
func (sf *ScriptFlow) createMapFromQuery(queryStr string, keyCols []string) (map[string]string, error) {
	result := make(map[string]string)
	query := sf.app.DB().NewQuery(queryStr)
	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		cols := make([]interface{}, len(keyCols)+1)
		cols[0] = &id
		for i := range keyCols {
			var col string
			cols[i+1] = &col
		}
		err := rows.Scan(cols...)
		if err != nil {
			return nil, err
		}
		keys := make([]string, len(keyCols))
		for i := range keyCols {
			keys[i] = *(cols[i+1].(*string))
		}
		result[strings.Join(keys, ",")] = id
	}
	return result, nil
}

func (sf *ScriptFlow) insertOrUpdate(table string, params dbx.Params, conflictColumns string, updateColumns ...string) error {
	setClause := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		setClause[i] = fmt.Sprintf("%s={:%s}", col, col)
	}

	query := sf.app.DB().NewQuery(fmt.Sprintf(
		`INSERT INTO %s (%s,created,updated) VALUES (%s,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP) ON CONFLICT (%s) DO UPDATE SET %s,updated=CURRENT_TIMESTAMP`,
		table,
		strings.Join(keys(params), ","),
		strings.Join(placeholders(params), ","),
		conflictColumns,
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
