# src/backend/Makefile
SHELL := /bin/bash
GO := go
DOCKER := docker
DOCKER_COMPOSE := docker-compose
GOLANGCI_LINT := golangci-lint
MOCKERY := mockery
SCRIPTS_DIR := ./scripts
BUILD_DIR := ./build
COVERAGE_DIR := ./coverage
ENV := development
CONFIG_FILE := ./config/$(ENV).yml
REGISTRY := document-mgmt
VERSION := latest
SERVICES := api worker

.PHONY: help
help: ## Display help information about available make targets
	@echo "Document Management Platform Makefile"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS := ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-30s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "For more information, see the README.md file."

.PHONY: setup
setup: ## Sets up the development environment
	@echo "Installing required Go tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.0 # v1.50.0+
	$(GO) install github.com/vektra/mockery/v2@v2.20.0 # v2.20.0+
	@echo "Creating necessary directories..."
	mkdir -p $(BUILD_DIR) $(COVERAGE_DIR)
	@echo "Running setup-dev.sh script..."
	$(SCRIPTS_DIR)/setup-dev.sh
	@echo "Starting required services using docker-compose..."
	$(DOCKER_COMPOSE) up -d

.PHONY: build
build: ## Builds the Go application
	@echo "Creating build directory if it doesn't exist..."
	mkdir -p $(BUILD_DIR)
	@echo "Building the application..."
	$(GO) build -o $(BUILD_DIR)/main ./cmd/...
	@echo "Build completed. Output binary: $(BUILD_DIR)/main"

.PHONY: clean
clean: ## Cleans build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	@echo "Build artifacts cleaned."

.PHONY: run
run: build ## Runs the application locally
	@echo "Running the application..."
	ENV=$(ENV) CONFIG_FILE=$(CONFIG_FILE) $(BUILD_DIR)/main

.PHONY: docker-build
docker-build: ## Builds Docker images
	@echo "Building Docker images..."
	$(SCRIPTS_DIR)/docker-build.sh

.PHONY: docker-push
docker-push: docker-build ## Pushes Docker images to registry
	@echo "Pushing Docker images..."
	$(SCRIPTS_DIR)/docker-push.sh

.PHONY: test
test: ## Runs all tests
	@echo "Running all tests..."
	$(SCRIPTS_DIR)/run-tests.sh

.PHONY: test-unit
test-unit: ## Runs unit tests
	@echo "Running unit tests..."
	$(SCRIPTS_DIR)/run-tests.sh -t unit

.PHONY: test-integration
test-integration: ## Runs integration tests
	@echo "Running integration tests..."
	$(SCRIPTS_DIR)/run-tests.sh -t integration

.PHONY: test-e2e
test-e2e: ## Runs end-to-end tests
	@echo "Running end-to-end tests..."
	$(SCRIPTS_DIR)/run-tests.sh -t e2e

.PHONY: coverage
coverage: ## Runs tests with coverage reporting
	@echo "Running tests with coverage reporting..."
	COVERAGE=true $(SCRIPTS_DIR)/run-tests.sh

.PHONY: lint
lint: ## Runs linting checks
	@echo "Running linting checks..."
	$(GOLANGCI_LINT) run ./...

.PHONY: fmt
fmt: ## Formats Go code
	@echo "Formatting Go code..."
	$(GO) fmt ./...

.PHONY: generate-mocks
generate-mocks: ## Generates mock implementations for testing
	@echo "Generating mock implementations..."
	$(SCRIPTS_DIR)/generate-mock.sh

.PHONY: migrate-up
migrate-up: ## Apply database migrations
	@echo "Applying database migrations..."
	$(SCRIPTS_DIR)/migration.sh up

.PHONY: migrate-down
migrate-down: ## Roll back database migrations
	@echo "Rolling back database migrations..."
	$(SCRIPTS_DIR)/migration.sh down

.PHONY: migrate-create
migrate-create: ## Create a new database migration
	@echo "Creating a new database migration..."
	$(SCRIPTS_DIR)/migration.sh create "$(name)"

.PHONY: docker-compose-up
docker-compose-up: ## Start all services using docker-compose
	@echo "Starting all services using docker-compose..."
	$(DOCKER_COMPOSE) up -d

.PHONY: docker-compose-down
docker-compose-down: ## Stop all services using docker-compose
	@echo "Stopping all services using docker-compose..."
	$(DOCKER_COMPOSE) down

.PHONY: docker-compose-logs
docker-compose-logs: ## Show logs from docker-compose services
	@echo "Showing logs from docker-compose services..."
	$(DOCKER_COMPOSE) logs -f