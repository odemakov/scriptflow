# ScriptFlow — Agent Reference

Distributed Command Scheduler with web interface. Executes commands across multiple nodes with scheduling, log handling, and notifications.

## Stack

| Layer | Tech |
|---|---|
| Backend | Go 1.25.0, PocketBase v0.39.3, gocron v2.19.1 |
| Database | SQLite (via PocketBase) |
| SSH | odemakov/sshrun |
| Frontend | Vue.js 3.5.34, TypeScript 5.9.3, Pinia, vue-router, @vueuse/core |
| UI | DaisyUI 4.12.24, TailwindCSS 3.4.19, ansi-to-html |
| Build | Vite 6.4.2 |

## Key Files

```
backend/main.go          # Entry point
backend/scriptflow.go    # Core scheduling logic
backend/api.go           # REST API endpoints
backend/types.go         # Data structures
backend/jobs.go          # Job management
backend/reconcile.go     # Reconcile logic
backend/schedule.go      # Scheduler wiring
backend/notification.go  # Slack notifications
backend/config.go        # Config parsing
backend/error.go         # Custom error types
frontend/src/            # Vue app (stores/, components/, lib/)
```

## Commands

Always use Makefile targets — they run in Docker:

```bash
make dev                 # Start full dev environment
make stop                # Stop stack
make clean               # Stop + remove volumes

make test                # All tests
make test-backend        # Go tests
make test-frontend       # Frontend tests

make lint-backend        # golangci-lint
make lint-frontend       # vue-tsc --noEmit

make build-frontend      # Production frontend build

# After backend code changes:
docker compose exec backend ./scriptflow reload
```

## Go Style

- Format: `gofmt` + `goimports`, lint: `golangci-lint`
- Naming: `PascalCase` exported, `camelCase` unexported, `ALL_CAPS` constants
- Interfaces: `-er` suffix (`Runner`, `Logger`)
- Struct tags: `snake_case` JSON/YAML
- Errors: custom types, wrap with `fmt.Errorf("context: %w", err)`
- Tests: testify/assert, table-driven, `_test.go` alongside source
- Patterns: dependency injection, `sync.Mutex` for shared state, `defer` for cleanup

## Commit Format

`Verb Noun` — present tense, capital first word: `Add X`, `Remove Y`, `Fix Z`, `Refactor A`, `Bump B`

## Dependency Updates

Go version tracks PocketBase — check PocketBase's go.mod before changing it.

```bash
# 1. Find latest PocketBase release
curl -s "https://api.github.com/repos/pocketbase/pocketbase/releases/latest" | python3 -c "import json,sys; print(json.load(sys.stdin)['tag_name'])"

# 2. Check what Go version it requires
curl -s "https://raw.githubusercontent.com/pocketbase/pocketbase/<version>/go.mod" | head -3

# 3. If Go version changed: update Dockerfile (2 FROM lines) and Makefile (test-backend line)
#    go.mod go directive updates automatically via go get

# 4. Bump PocketBase (run from repo root — mounts backend/ into container)
docker run --rm -v $(pwd)/backend:/app -w /app golang:1.25-alpine go get github.com/pocketbase/pocketbase@<version>
docker run --rm -v $(pwd)/backend:/app -w /app golang:1.25-alpine go mod tidy
docker run --rm -v $(pwd)/backend:/app -w /app golang:1.25-alpine go build ./...
```

Update the stack table in this file after bumping.

## Constraints

- Never use `npx` or `npm` directly — always Makefile targets
- Establish design agreement before writing code
- No speculative abstractions or features beyond the request
- Touch only what the task requires
