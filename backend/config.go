package main

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Projects []ConfigProject `yaml:"projects"`
	Nodes    []ConfigNode    `yaml:"nodes"`
	Tasks    []ConfigTask    `yaml:"tasks"`
}

type ConfigProject struct {
	Name string `yaml:"name"`
	Slug string `yaml:"slug"`
}

type ConfigNode struct {
	Host       string `yaml:"host"`
	Username   string `yaml:"username"`
	PrivateKey string `yaml:"private_key"`
}

type ConfigTask struct {
	Name            string `yaml:"name"`
	Slug            string `yaml:"slug"`
	Command         string `yaml:"command"`
	Schedule        string `yaml:"schedule"`
	Node            string `yaml:"node"`
	Project         string `yaml:"project"`
	Active          bool   `yaml:"active"`
	PrependDatetime bool   `yaml:"prepend_datetime"`
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
	return nil
}

func (sf *ScriptFlow) updateFromConfigProject() {
	// insert or update projects
	for _, project := range sf.config.Projects {
		// skip empty name or slug
		if project.Name == "" || project.Slug == "" {
			sf.app.Logger().Warn("[config] project name or slug is empty", slog.Any("project", project))
			continue
		}
		err := sf.insertOrUpdate("projects", dbx.Params{
			"slug": project.Slug,
			"name": project.Name,
		}, "slug", "name")
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
			sf.app.Logger().Warn("[config] node host or username is empty", slog.Any("node", node))
			continue
		}
		err := sf.insertOrUpdate("nodes", dbx.Params{
			"host":        node.Host,
			"username":    node.Username,
			"private_key": node.PrivateKey,
		}, "host,username", "private_key")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update node", slog.Any("error", err))
		}
	}
}

func (sf *ScriptFlow) updateFromConfigTaks() {
	projects, err := sf.createMapFromQuery("SELECT id, slug FROM projects", []string{"name"})
	if err != nil {
		sf.app.Logger().Error("[config] failed to create projects map", slog.Any("error", err))
		return
	}

	nodes, err := sf.createMapFromQuery("SELECT id, host, username FROM nodes", []string{"host", "username"})
	if err != nil {
		sf.app.Logger().Error("[config] failed to create nodes map", slog.Any("error", err))
		return
	}

	// insert or update tasks
	for _, task := range sf.config.Tasks {
		// skip empty name, slug, command, schedule, node, project
		if task.Name == "" || task.Slug == "" || task.Command == "" || task.Schedule == "" || task.Node == "" || task.Project == "" {
			sf.app.Logger().Warn("[config] task name, slug, command, schedule, node or project is empty", slog.Any("task", task))
			continue
		}
		// check node exists in the map
		if _, ok := nodes[task.Node]; !ok {
			sf.app.Logger().Warn("[config] node not found for task", slog.Any("task", task))
			continue
		}
		// check project exists in the map
		if _, ok := projects[task.Project]; !ok {
			sf.app.Logger().Warn("[config] project not found for task", slog.Any("task", task))
			continue
		}
		err := sf.insertOrUpdate("tasks", dbx.Params{
			"slug":             task.Slug,
			"name":             task.Name,
			"command":          task.Command,
			"schedule":         task.Schedule,
			"node":             nodes[task.Node],
			"project":          projects[task.Project],
			"active":           task.Active,
			"prepend_datetime": task.PrependDatetime,
		}, "slug", "name", "command", "schedule", "node", "project", "active", "prepend_datetime")
		if err != nil {
			sf.app.Logger().Error("[config] failed to insert or update task", slog.Any("error", err))
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
