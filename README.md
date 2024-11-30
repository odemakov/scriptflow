# Scriptflow is yet another script framework

**Scriptflow** is a lightweight, one-file solution designed to simplify script management, execution, monitoring and alerting(TODO). It provides an easy-to-use interface for running scripts, collecting and viewing logs and exit statuses. Based on PocketBase v0.23 with simple Vue app as UI embeded in sinlge binary file.

## Features

- Simple script execution and monitoring
- Log collection and management
- Exit status tracking
- Easy to install and use
- Web-based interface
- REST API

## Architecture

**Scriptflow** consists of two main components:

- Backend: Built on top of [Pocketbase](https://pocketbase.io/) for robust data storage and API
- Frontend: Modern UI built with [Vue.js](https://vuejs.org/)

## Quick Start

TODO

### Using Docker (Recommended)

1. Clone the repository: `git clone https://github.com/odemakov/scriptflow`
2. Build `cd scriptflow && make build && make extract`

### Cron scheduling

Task.schedule field parsed by [robfig/cron/v3](https://pkg.go.dev/github.com/robfig/cron/v3) library

### Development

Development environment completely inside Docker with autorestart backend and frotend apps on file changes.

`make dev`
