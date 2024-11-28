# Makefile

.PHONY: build dev test clean extract

DOCKER_COMPOSE=docker compose
DOCKER=docker
PROJECT_NAME=scriptflow
BUILD_OUTPUT=backend/src/scriptflow

# Build production-ready image and extract executable
build:
	$(DOCKER) build --target app -t $(PROJECT_NAME):prod .

# Run development environment
dev:
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml --env-file .env.development up --build

# Run unit tests for frontend and backend
test:
	$(DOCKER) run --rm -v $(PWD)/backend:/app -w /app golang:1.23-alpine go test ./...
	$(DOCKER) run --rm -v $(PWD)/frontend:/app -w /app node:alpine npm test

# Stop all containers and clean up
clean:
	$(DOCKER_COMPOSE) down --volumes --remove-orphans

# Extract built Go executable from the production image
extract:
	$(DOCKER) create --name temp-container $(PROJECT_NAME):prod
	$(DOCKER) cp temp-container:/app/scriptflow $(BUILD_OUTPUT)
	$(DOCKER) rm -f temp-container
