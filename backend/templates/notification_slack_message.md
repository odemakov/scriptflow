{{if eq .Item.Status "completed"}}✅{{else}}❌{{end}} *{{ .Subject }}*

Task {{.TaskName}} finished with status `{{.Item.Status}}`

{{.TaskUrl}}
{{.RunUrl}}

* Command: `{{.Item.Command}}`
* Host: `{{.Item.Host}}`
* Status: `{{.Item.Status}}`
{{if .Item.Error}}* Error: {{.Item.Error}}{{end}}
* Exit code: `{{.Item.ExitCode}}`
* Created: `{{.Item.Created}}`
* Updated: `{{.Item.Updated}}`