.PHONY: help build run test clean docker-build docker-up docker-down docker-logs lint format deps dev-history migrate test-api install-deps

BINARY_NAME=prosig
DOCKER_IMAGE=prosig
DOCKER_TAG=latest
PORT?=8080
AUTH_USERNAME?=admin
AUTH_PASSWORD?=secret
DATABASE_URL?=postgres://prosig:prosig_pass@localhost:5432/prosig?sslmode=disable

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-deps: ## Install Go dependencies
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/prosig

run: ## Run the application locally
	@echo "Running $(BINARY_NAME) on port $(PORT)..."
	PORT=$(PORT) AUTH_USERNAME=$(AUTH_USERNAME) AUTH_PASSWORD=$(AUTH_PASSWORD) DATABASE_URL=$(DATABASE_URL) go run ./cmd/prosig

run-in-memory: ## Run the application with in-memory storage (no database)
	@echo "Running $(BINARY_NAME) with in-memory storage..."
	PORT=$(PORT) AUTH_USERNAME=$(AUTH_USERNAME) AUTH_PASSWORD=$(AUTH_PASSWORD) DATABASE_URL="" go run ./cmd/prosig

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

format: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...
	@echo "Code formatted"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	docker-compose up -d
	@echo "Services started. Use 'make docker-logs' to view logs."

docker-down: ## Stop services with docker-compose
	@echo "Stopping services..."
	docker-compose down
	@echo "Services stopped."

docker-logs: ## View docker-compose logs
	docker-compose logs -f

docker-restart: ## Restart docker-compose services
	@echo "Restarting services..."
	docker-compose restart
	@echo "Services restarted."

docker-clean: ## Stop and remove containers, volumes, and images
	@echo "Cleaning Docker resources..."
	docker-compose down -v --rmi local
	@echo "Docker resources cleaned."

migrate: ## Run database migrations
	@echo "Running database migrations..."
	@if [ ! -f "migrations/001_init.sql" ]; then \
		echo "Warning: migrations/001_init.sql not found. Creating migrations directory..."; \
		mkdir -p migrations; \
	fi
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "Error: DATABASE_URL not set. Use: make migrate DATABASE_URL=postgres://user:pass@host:port/dbname"; \
		exit 1; \
	fi
	@which psql > /dev/null || (echo "psql not found. Please install PostgreSQL client tools." && exit 1)
	@if [ -f "migrations/001_init.sql" ]; then \
		psql $(DATABASE_URL) -f migrations/001_init.sql; \
	else \
		echo "No migration file found. Skipping migration."; \
	fi

migrate-docker: ## Run database migrations in Docker container
	@echo "Running database migrations in Docker..."
	@if docker-compose ps db 2>/dev/null | grep -q "Up"; then \
		docker-compose exec -T db psql -U prosig -d prosig < migrations/001_init.sql 2>/dev/null || \
		echo "Migrations should run automatically on first container start"; \
	else \
		echo "Database container is not running. Start with: make docker-up"; \
	fi

db-shell: ## Connect to PostgreSQL database shell
	@echo "Connecting to database..."
	docker-compose exec db psql -U prosig -d prosig

test-api: ## Run API tests using the test script
	@echo "Running API tests..."
	@which http > /dev/null || (echo "httpie not found. Install with: pip install httpie" && exit 1)
	@if ! curl -s http://localhost:$(PORT)/metrics > /dev/null 2>&1; then \
		echo "Error: Server is not running on port $(PORT)"; \
		echo "Start the server first with: make run"; \
		exit 1; \
	fi
	./test_api.sh


setup: install-deps migrate ## Initial setup: install deps and run migrations
	@echo "Setup complete!"

dev: ## Start development environment (docker-compose + hot reload)
	@echo "Starting development environment..."
	@echo "Make sure you have docker-compose running: make docker-up"
	@echo "Then run the app with auto-reload: make run"

install: build ## Install binary to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installed successfully!"

uninstall: ## Remove binary from /usr/local/bin
	@echo "Removing $(BINARY_NAME) from /usr/local/bin..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Removed successfully!"

check: lint test ## Run checks (lint + test)

ci: clean install-deps lint test build docker-build ## CI pipeline: clean, deps, lint, test, build, docker
	@echo "CI pipeline completed successfully!"

all: clean install-deps build test docker-build ## Run all: clean, deps, build, test, docker
	@echo "All targets completed!"

.DEFAULT_GOAL := help
