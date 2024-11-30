# Makefile

.PHONY: build dev test clean extract

DOCKER_COMPOSE=docker compose
DOCKER=docker
PROJECT_NAME=scriptflow
BUILD_OUTPUT=scriptflow

# Build production-ready image and extract executable
#build:
#	DOCKER_BUILDKIT=1 $(DOCKER) build --no-cache --target app -t $(PROJECT_NAME):prod .

# Run development environment
dev:
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml up --build

# Create migration file
create_migration_snapshot:
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml exec backend go run . migrate history-sync
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml exec backend go run . migrate collections

# Stop dev stack
stop:
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml stop

# Run unit tests for frontend and backend
test: _test_backend _test_frontend
_test_backend:
	@echo $(DOCKER) run --rm -v $(PWD)/backend:/app -w /app golang:1.23-alpine go test ./...
_test_frontend:
	$(DOCKER) build --no-cache --target builder-frontend -t $(PROJECT_NAME):builder-frontend .
	$(DOCKER) run --rm $(PROJECT_NAME):builder-frontend npm run test

# Stop all containers and clean up
clean:
	$(DOCKER_COMPOSE) down --volumes --remove-orphans

# Extract built Go executable from the production image
extract:
	$(DOCKER) create --name temp-container $(PROJECT_NAME):prod
	$(DOCKER) cp temp-container:/app/scriptflow $(BUILD_OUTPUT)
	$(DOCKER) rm -f temp-container
