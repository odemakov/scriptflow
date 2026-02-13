# ScriptFlow

Distributed command scheduler with a web UI. Run scripts across multiple nodes on a schedule, collect logs in one place, get notified when things break.

Built on [PocketBase](https://pocketbase.io) — ships as a single binary. Two folders (database + logs) and you're done.

## Why not just use...

- **Cron** — no central management, no UI, no notifications
- **Jenkins** — way too heavy for "run this script every hour"
- **Airflow** — great for data pipelines, overkill for ops tasks
- **Ansible** — config management tool, not a scheduler
- **K8s CronJobs** — great if you're running Kubernetes

## What you get

- Script execution and monitoring from a single dashboard
- Centralized log collection
- Real-time task status tracking
- Email and Slack notifications
- Everything PocketBase gives you (auth, realtime, API, admin UI)
- 5-minute setup

## Quick Start

```bash
mkdir scriptflow && cd scriptflow
wget https://github.com/odemakov/scriptflow/releases/download/v0.0.4/scriptflow_Linux_x86_64.tar.gz
tar -xzf scriptflow_Linux_x86_64.tar.gz
./scriptflow --http 0.0.0.0:8090 --dev serve
```

You can also define projects, nodes, tasks and notifications in a config file — see `config-example.yml`.

```bash
./scriptflow --http 0.0.0.0:8090 --dev --config config-example.yml serve
```

## Running as a systemd service

Create `/etc/systemd/system/scriptflow.service`:

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

Then enable and start it:

```bash
systemctl daemon-reload && systemctl enable scriptflow.service && systemctl start scriptflow.service
```

## Scheduling

The `Task.schedule` field accepts:

**Cron expressions** — `0 * * * *` (top of every hour), the usual.

**Duration strings** — `@every 1h30m` runs every 1.5 hours. Has ±10% jitter built in to spread load. See [time.ParseDuration](https://pkg.go.dev/time#ParseDuration) for the format.

**Jenkins-style H (hash)** — distributes tasks across a time range:

```
H * * * *         # hourly, at a consistent but spread-out minute
H(0-30) * * * *   # hourly, somewhere in the first 30 minutes
H H(0-6) * * *    # daily, somewhere between midnight and 6 AM
```

The hash is deterministic per task ID — same task always fires at the same time, but different tasks get spread out. Avoids the thundering herd problem.

## Development

Everything runs in Docker with auto-restart on file changes:

```bash
make dev
```
