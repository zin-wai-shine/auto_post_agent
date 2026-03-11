# Super-Agent CLI Makefile

APP_NAME := super-agent
BUILD_DIR := ./build
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './ui/*')

.PHONY: all build run clean test lint db-up db-down db-reset migrate help

## help: Show this help message
help:
	@echo "Super-Agent CLI — Available Commands:"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'

## build: Build the CLI binary
build:
	@echo "🔨 Building $(APP_NAME)..."
	go build -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "✅ Built: $(BUILD_DIR)/$(APP_NAME)"

## run: Build and run the CLI
run: build
	$(BUILD_DIR)/$(APP_NAME)

## clean: Remove build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	@echo "✅ Clean"

## test: Run all tests
test:
	go test -v -race ./...

## lint: Run linter
lint:
	golangci-lint run ./...

## db-up: Start PostgreSQL via Docker Compose
db-up:
	@echo "🐘 Starting PostgreSQL..."
	docker compose up -d
	@echo "✅ Database ready at localhost:5433"

## db-down: Stop PostgreSQL
db-down:
	docker compose down

## db-reset: Destroy and recreate the database
db-reset: db-down
	docker volume rm bolt_haven_cli_pgdata 2>/dev/null || true
	$(MAKE) db-up

## migrate: Run database migrations
migrate: build
	$(BUILD_DIR)/$(APP_NAME) migrate --db-url "postgres://superagent:superagent_dev_2024@localhost:5433/superagent?sslmode=disable"

## deps: Install Go dependencies
deps:
	go mod tidy
	go mod download

## dev: Start development (DB + build)
dev: db-up deps build
	@echo "🚀 Development environment ready!"
	@echo "   CLI: $(BUILD_DIR)/$(APP_NAME) --help"
	@echo "   DB:  postgres://superagent:superagent_dev_2024@localhost:5432/superagent"
