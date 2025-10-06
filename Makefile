.PHONY: help build run test clean docker-up docker-down migrate

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Собрать приложение
	@echo "Building application..."
	@go build -o bin/blog-api cmd/api/main.go

run: ## Запустить приложение
	@echo "Starting application..."
	@go run cmd/api/main.go

test: ## Запустить тесты
	@echo "Running tests..."
	@go test -v ./...

clean: ## Очистить собранные файлы
	@echo "Cleaning..."
	@rm -rf bin/

deps: ## Установить зависимости
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

docker-up: ## Запустить PostgreSQL в Docker
	@echo "Starting PostgreSQL container..."
	@docker run --name blog-postgres -e POSTGRES_PASSWORD=password -e POSTGRES_USER=bloguser -e POSTGRES_DB=blogdb -p 5432:5432 -d postgres:15-alpine

docker-down: ## Остановить PostgreSQL контейнер
	@echo "Stopping PostgreSQL container..."
	@docker stop blog-postgres
	@docker rm blog-postgres

docker-logs: ## Показать логи PostgreSQL
	@docker logs -f blog-postgres

migrate-up: ## Применить миграции
	@echo "Running migrations..."
	@go run cmd/api/main.go

lint: ## Запустить линтер
	@echo "Running linter..."
	@golangci-lint run

format: ## Форматировать код
	@echo "Formatting code..."
	@go fmt ./...

dev: docker-up ## Запустить dev окружение
	@sleep 2
	@make run

.DEFAULT_GOAL := help