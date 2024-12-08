# Scriptflow is a Distributed Command Scheduler with web interface

**ScriptFlow** is a Distributed Command Scheduler designed to manage and execute commands across multiple nodes with customizable scheduling. It handles logs efficiently and includes notifications to keep users updated on task statuses and results.

ScriptFlow is easy to install and maintain, built on a lightweight [PocketBase](https://pocketbase.io) framework. The entire system is contained in a single file, with only two additional folders: one for the database and one for logs. This simplicity makes setup quick and ensures users can manage and monitor tasks without the hassle of complex systems.

# Existing systems

- **Cron**: A traditional Unix-based job scheduler, Cron is powerful but lacks centralized management, web interface, and notification capabilities.
- **Jenkins**: Primarily a CI/CD tool, Jenkins can schedule tasks across nodes but is heavyweight and complex to set up for simpler scheduling needs.
- **Airflow**: Apache Airflow excels at orchestrating workflows but requires significant resources and knowledge to install and manage.
- **Ansible**: Often used for configuration management and ad-hoc command execution, it doesn’t focus on recurring job scheduling.
- **Kubernetes CronJobs**: Built for containerized environments, it’s complex and overkill for simpler scheduling needs outside Kubernetes.

# Features

- easy script execution and monitoring
- centralized log collection and management
- real-time tracking of task statuses and outcomes
- quick and hassle-free installation
- user-friendly web interface
- all bells and whistles come with [PocketBase](https://pocketbase.io)

# Quick Start

Create project directory `mkdir scriptflow && cd scriptflow`

Download release `wget https://github.com/odemakov/scriptflow/releases/download/v0.0.4/scriptflow_Linux_x86_64.tar.gz`

Extract it `tar -xzf scriptflow_Linux_x86_64.tar.gz`

Run `./scriptflow --http 0.0.0.0:8090 --dev serve`

# Run as system service

Create `/etc/systemd/system/scriptflow.service` file

```
[Unit]
Description = scriptflow

[Service]
Type           = simple
User           = scriptflow
Group          = scriptflow
LimitNOFILE    = 4096
Restart        = always
RestartSec     = 5s
StandardOutput = append:/var/log/scriptflow-out.log
StandardError  = append:/var/log/scriptflow-err.log
ExecStart      = /data/scriptflow/scriptflow --dir /data/scriptflow/data/pb_data --http 0.0.0.0:8090 serve

[Install]
WantedBy = multi-user.target
```

Enable and restart service

`systemctl daemon-reload && systemctl enable scriptflow.service && systemctl start scriptflow.service`

# Cron scheduling

Task.schedule field parsed by [robfig/cron/v3](https://pkg.go.dev/github.com/robfig/cron/v3) library

# Development

Development environment completely inside Docker with autorestart backend and frotend apps on file changes.

`make dev`
