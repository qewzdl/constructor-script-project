.PHONY: help build run test clean deps docker-up docker-down docker-logs migrate-up lint format dev watch db-reset

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m%s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building application..."
	@go build -o bin/blog-api cmd/api/main.go

run: ## Run the application
	@echo "Starting application..."
	@go run cmd/api/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/

deps: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

docker-up: ## Start PostgreSQL in Docker
	@echo "Starting PostgreSQL container..."
	@docker run --name blog-postgres -e POSTGRES_PASSWORD=devpassword -e POSTGRES_USER=devuser -e POSTGRES_DB=constructor -p 5432:5432 -d postgres:15-alpine

docker-down: ## Stop PostgreSQL container
	@echo "Stopping PostgreSQL container..."
	@docker stop blog-postgres || true
	@docker rm blog-postgres || true

docker-logs: ## Show PostgreSQL logs
	@docker logs -f blog-postgres

migrate-up: ## Run migrations
	@echo "Running migrations..."
	@go run cmd/api/main.go

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

format: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

dev: docker-up ## Start development environment
	@sleep 2
	@make run

watch: ## Run app with auto-reload (requires Air)
	@if ! command -v air >/dev/null 2>&1; then \
		echo "\033[31mAir is not installed. Install it with:\033[0m go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi
	@air

db-reset: ## Fully reset PostgreSQL with a clean database
	@echo "Resetting PostgreSQL database..."
	@docker compose down -v --remove-orphans
	@docker compose up -d postgres

.DEFAULT_GOAL := help
