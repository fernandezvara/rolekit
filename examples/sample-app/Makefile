.PHONY: help start stop run test clean

# Default database configuration
DATABASE_URL ?= postgres://postgres:password@localhost:5418/rolekit_test?sslmode=disable

help: ## Show this help message
	@echo "RoleKit Sample Application"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  help     Show this help message"
	@echo "  start    Start PostgreSQL 18 container"
	@echo "  stop     Stop PostgreSQL 18 container"
	@echo "  run      Run the sample application"
	@echo "  test     Run the sample application tests"
	@echo "  clean    Stop container and clean up"

start: ## Start PostgreSQL 18 container
	@echo "🚀 Starting PostgreSQL 18..."
	podman compose up -d
	@echo "⏳ Waiting for database to be ready..."
	@until podman exec rolekit-postgres-18 pg_isready -U postgres 2>/dev/null; do sleep 1; done
	@echo "✅ PostgreSQL 18 is ready!"

stop: ## Stop PostgreSQL 18 container
	@echo "🛑 Stopping PostgreSQL 18..."
	podman compose down
	@echo "✅ PostgreSQL 18 stopped"

run: ## Run the sample application
	@echo "🚀 Running RoleKit Sample Application..."
	DATABASE_URL=$(DATABASE_URL) go run .

test: ## Run the sample application with test database
	@echo "🧪 Testing RoleKit Sample Application..."
	DATABASE_URL=$(DATABASE_URL) go run .

clean: ## Stop container and clean up
	@echo "🧹 Cleaning up..."
	podman compose down -v --remove-orphans
	@echo "✅ Cleanup complete"

# Development targets
dev-setup: start ## Set up development environment
	@echo "🔧 Setting up development environment..."
	@echo "✅ Development environment ready"

dev-run: ## Run in development mode
	@echo "🔧 Running in development mode..."
	DATABASE_URL=$(DATABASE_URL) go run .

dev-test: ## Test in development mode
	@echo "🧪 Testing in development mode..."
	DATABASE_URL=$(DATABASE_URL) go run .
