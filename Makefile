# OVN Control Platform Makefile

# Variables
GO := go
NPM := npm
DOCKER := docker
DOCKER_COMPOSE := docker-compose
HELM := helm
MIGRATE := migrate

# Build variables
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Directories
API_DIR := .
WEB_DIR := ./web
CHARTS_DIR := ./charts/ovncp
TEST_DIR := ./test
COVERAGE_DIR := ./coverage

# Binary name
BINARY_NAME := ovncp
MAIN_PATH := cmd/ovncp/main.go

# Targets
.PHONY: all build test clean help

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## all: Build everything
all: build-api build-web

## build: Build both API and web
build: build-api build-web

## build-api: Build the API server
build-api:
	@echo "Building API server..."
	$(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)

## build-web: Build the web UI
build-web:
	@echo "Building web UI..."
	cd $(WEB_DIR) && $(NPM) run build

## test: Run all tests
test: test-unit test-integration

## test-unit: Run unit tests with coverage
test-unit:
	@echo "Running unit tests..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -v -race -coverprofile=$(COVERAGE_DIR)/unit.out -covermode=atomic ./internal/...
	$(GO) test -v -race -coverprofile=$(COVERAGE_DIR)/cmd.out -covermode=atomic ./cmd/...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -v -race -tags=integration -coverprofile=$(COVERAGE_DIR)/integration.out -covermode=atomic ./test/integration/...

## test-e2e: Run end-to-end tests
test-e2e:
	@echo "Running end-to-end tests..."
	cd $(TEST_DIR)/e2e && $(GO) test -v -tags=e2e ./...

## test-coverage: Generate test coverage report
test-coverage: test-unit
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	@echo "mode: set" > $(COVERAGE_DIR)/coverage.out
	@tail -n +2 $(COVERAGE_DIR)/unit.out >> $(COVERAGE_DIR)/coverage.out 2>/dev/null || true
	@tail -n +2 $(COVERAGE_DIR)/cmd.out >> $(COVERAGE_DIR)/coverage.out 2>/dev/null || true
	@tail -n +2 $(COVERAGE_DIR)/integration.out >> $(COVERAGE_DIR)/coverage.out 2>/dev/null || true
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

## benchmark: Run performance benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem -run=^# ./...

## lint: Run linters
lint: lint-go lint-web

## lint-go: Run Go linters
lint-go:
	@echo "Running Go linters..."
	golangci-lint run ./...

## lint-web: Run web linters
lint-web:
	@echo "Running web linters..."
	cd $(WEB_DIR) && $(NPM) run lint

## fmt: Format code
fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...
	@echo "Formatting web code..."
	cd $(WEB_DIR) && $(NPM) run format

## deps: Install dependencies
deps: deps-go deps-web

## deps-go: Install Go dependencies
deps-go:
	@echo "Installing Go dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## deps-web: Install web dependencies
deps-web:
	@echo "Installing web dependencies..."
	cd $(WEB_DIR) && $(NPM) ci

## docker-build: Build Docker images
docker-build:
	@echo "Building Docker images..."
	$(DOCKER) build -t ovncp-api:$(VERSION) -f Dockerfile .
	$(DOCKER) build -t ovncp-web:$(VERSION) -f web/Dockerfile ./web

## docker-compose-up: Start services with docker-compose
docker-compose-up:
	@echo "Starting services with docker-compose..."
	$(DOCKER_COMPOSE) up -d

## docker-compose-down: Stop services with docker-compose
docker-compose-down:
	@echo "Stopping services with docker-compose..."
	$(DOCKER_COMPOSE) down

## docker-run: Run the application
docker-run: docker-build
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env ovncp-api:$(VERSION)

## dev: Run development environment
dev:
	@echo "Starting development environment..."
	$(MAKE) deps
	$(DOCKER_COMPOSE) -f docker-compose.yml up -d postgres
	@echo "Starting API server..."
	$(GO) run $(MAIN_PATH)

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf $(COVERAGE_DIR)/
	rm -rf $(WEB_DIR)/dist/
	$(GO) clean -cache -testcache

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	cd $(WEB_DIR) && $(NPM) install -g playwright

## generate: Generate code (swagger, mocks, etc)
generate:
	@echo "Generating code..."
	$(GO) generate ./...
	swag init -g $(MAIN_PATH) -o docs/swagger

## security: Run security scans
security:
	@echo "Running security scans..."
	@echo "Checking Go dependencies..."
	@go list -json -m all | nancy sleuth

# Default target
.DEFAULT_GOAL := help