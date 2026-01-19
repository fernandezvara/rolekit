# Makefile for dbkit - PostgreSQL integration testing
# Supports both Docker and Podman container runtimes

.PHONY: help detect-runtime start stop test clean lint

# Detect container runtime (docker or podman)
DOCKER := $(shell command -v docker 2> /dev/null)
PODMAN := $(shell command -v podman 2> /dev/null)

ifdef DOCKER
    CONTAINER_RUNTIME := docker
else ifdef PODMAN
    CONTAINER_RUNTIME := podman
else
    $(error No container runtime found. Please install docker or podman)
endif

# Detect compose tool (docker-compose, docker compose, or podman-compose)
DOCKER_COMPOSE := $(shell command -v docker-compose 2> /dev/null)
DOCKER_COMPOSE_PLUGIN := $(shell docker compose version 2> /dev/null && echo "docker compose")
PODMAN_COMPOSE := $(shell command -v podman-compose 2> /dev/null)

ifdef DOCKER_COMPOSE
    COMPOSE_CMD := docker-compose
else ifdef DOCKER_COMPOSE_PLUGIN
    COMPOSE_CMD := docker compose
else ifdef PODMAN_COMPOSE
    COMPOSE_CMD := podman-compose
else
    $(error No compose tool found. Please install docker-compose, docker compose plugin, or podman-compose)
endif

# Default test timeout
TEST_TIMEOUT := 5m

# Database URL for testing
TEST_DATABASE_URL := postgres://postgres:password@localhost:5418/rolekit_test?sslmode=disable

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "dbkit - PostgreSQL Integration Testing"
	@echo ""
	@echo "Detected runtime: $(CONTAINER_RUNTIME)"
	@echo "Detected compose: $(COMPOSE_CMD)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  help                 Show this help message"
	@echo "  detect-runtime       Show runtime and compose tool"
	@echo "  start                Start PostgreSQL database"
	@echo "  stop                 Stop PostgreSQL database"
	@echo "  test                 Run unit tests (no database)"
	@echo "  test-all             Run all tests including database tests"
	@echo "  test-coverage        Run tests with coverage"
	@echo "  bench                Run benchmark tests"
	@echo "  lint                 Run golangci-lint"
	@echo "  clean                Clean up containers and volumes"

detect-runtime: ## Show detected container runtime and compose tool
	@echo "Container runtime: $(CONTAINER_RUNTIME)"
	@echo "Compose command: $(COMPOSE_CMD)"

start: ## Start PostgreSQL database for testing
	@echo "$(GREEN)Starting PostgreSQL 18...$(NC)"
	$(COMPOSE_CMD) -f docker-compose.yml up -d
	@echo "$(YELLOW)Waiting for database to be ready...$(NC)"
	@timeout 60 bash -c 'until $(CONTAINER_RUNTIME) exec rolekit-postgres-18 pg_isready -U postgres; do sleep 1; done'
	@echo "$(GREEN)PostgreSQL is ready!$(NC)"
	@echo "$(YELLOW)Database URL: $(TEST_DATABASE_URL)$(NC)"

stop: ## Stop PostgreSQL database
	@echo "$(YELLOW)Stopping PostgreSQL...$(NC)"
	$(COMPOSE_CMD) -f docker-compose.yml down
	@echo "$(GREEN)PostgreSQL stopped.$(NC)"

test: ## Run unit tests (no database required)
	@echo "$(GREEN)Running unit tests...$(NC)"
	go test -v -cover -race -timeout $(TEST_TIMEOUT) ./...

test-all: start ## Run all tests including database tests
	@echo "$(GREEN)Running all tests with database...$(NC)"
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v -cover -race -timeout $(TEST_TIMEOUT) ./...

# Run tests with coverage
test-coverage: start ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" \
		go test -v -race -coverprofile=coverage.out -timeout $(TEST_TIMEOUT) ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# Benchmark tests
bench: start ## Run benchmark tests
	@echo "$(GREEN)Running benchmarks against PostgreSQL...$(NC)"
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" \
		go test -v -bench=. -benchmem -timeout $(TEST_TIMEOUT) ./...

# Lint code with golangci-lint
lint: ## Run golangci-lint
	@echo "$(GREEN)Checking for golangci-lint installation...$(NC)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(RED)Error: golangci-lint is not installed.$(NC)"; \
		echo "$(YELLOW)Please install golangci-lint from: https://golangci-lint.run/docs/welcome/install/local/$(NC)"; \
		echo "$(YELLOW)Or run: go install github.com/golangci/golang-lint/v2/cmd/golangci-lint@latest$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Running golangci-lint...$(NC)"
	golangci-lint run ./...
	@echo "$(GREEN)Linting complete!$(NC)"

clean: ## Clean up containers and volumes
	@echo "$(YELLOW)Cleaning up containers and volumes...$(NC)"
	$(COMPOSE_CMD) -f docker-compose.yml down -v
	@echo "$(GREEN)Cleanup complete.$(NC)"
