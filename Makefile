.PHONY: all build run test lint clean proto docker-up docker-down help

# Variables
BINARY_NAME=linktor
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"

# Default target
all: lint build

## Build

build: ## Build the server binary
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/server

build-cli: ## Build the CLI binary
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/msgfy ./cmd/cli

## Run

run: ## Run the server
	$(GO) run ./cmd/server

run-dev: docker-up ## Run in development mode with Docker services
	$(GO) run ./cmd/server

## Test

test: ## Run tests
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## Lint

lint: ## Run linter
	golangci-lint run

lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix

## Protobuf

proto: ## Generate protobuf code
	buf generate

proto-lint: ## Lint protobuf files
	buf lint

proto-breaking: ## Check for breaking changes
	buf breaking --against '.git#branch=main'

## Docker

docker-up: ## Start Docker services
	docker-compose up -d

docker-down: ## Stop Docker services
	docker-compose down

docker-logs: ## Show Docker logs
	docker-compose logs -f

docker-clean: ## Remove Docker volumes
	docker-compose down -v

## Database

db-reset: ## Reset database
	docker-compose down -v
	docker-compose up -d postgres
	sleep 5
	docker-compose exec -T postgres psql -U linktor -d linktor -f /docker-entrypoint-initdb.d/init.sql

## Dependencies

deps: ## Download dependencies
	$(GO) mod download

deps-tidy: ## Tidy dependencies
	$(GO) mod tidy

deps-update: ## Update dependencies
	$(GO) get -u ./...
	$(GO) mod tidy

## Clean

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf coverage.out coverage.html
	rm -rf internal/grpc/gen/

## Install tools

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

## Help

help: ## Show this help
	@echo "Linktor - Multichannel Messaging Platform"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
