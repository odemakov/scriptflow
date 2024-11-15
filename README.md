# Script-flow is yet another script framework

**Script-flow** is a lightweight framework designed to simplify script management, execution monitoring and alerting(todo). It provides an easy-to-use interface for running scripts, collecting logs and exit statuses.

## Features

- Simple script execution and monitoring
- Log collection and management
- Exit status tracking
- Easy to install and use
- Web-based interface
- REST API

## Architecture

**Script-flow** consists of two main components:

- Backend: Built on top of [Pocketbase](https://pocketbase.io/) for robust data storage and API
- Frontend: Modern UI built with [Vue.js](https://vuejs.org/)

## Quick Start

### Using Docker (Recommended)

1. Clone the repository:

### Cron scheduling

Task.schedule field parsed by [robfig/cron/v3](https://pkg.go.dev/github.com/robfig/cron/v3) library

### Development

`cd scc/backend`

`go run main.go --dir ../data/ serve`
