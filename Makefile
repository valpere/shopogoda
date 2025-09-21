# Variables
BINARY_NAME=shopogoda
DOCKER_IMAGE=shopogoda
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Colors for output
CYAN=\033[0;36m
GREEN=\033[0;32m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: help build run test clean docker-build docker-up docker-down deps lint

help: ## Show this help message
    @echo "$(CYAN)ShoPogoda (Що Погода) - Development Commands$(NC)"
    @echo ""
    @awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "$(GREEN)%-15s$(NC) %s\n", $1, $2}' $(MAKEFILE_LIST)

deps: ## Install Go dependencies
    @echo "$(CYAN)Installing dependencies...$(NC)"
    @go mod download
    @go mod tidy

build: deps ## Build the application
    @echo "$(CYAN)Building $(BINARY_NAME)...$(NC)"
    @go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/bot/main.go

run: build ## Run the application
    @echo "$(CYAN)Running $(BINARY_NAME)...$(NC)"
    @./bin/$(BINARY_NAME)

test: ## Run tests
    @echo "$(CYAN)Running tests...$(NC)"
    @go test -v ./...

test-coverage: ## Run tests with coverage
    @echo "$(CYAN)Running tests with coverage...$(NC)"
    @go test -coverprofile=coverage.out ./...
    @go tool cover -html=coverage.out -o coverage.html
    @echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

test-integration: ## Run integration tests
    @echo "$(CYAN)Running integration tests...$(NC)"
    @go test -tags=integration -v ./tests/integration/...

test-e2e: ## Run end-to-end tests
    @echo "$(CYAN)Running E2E tests...$(NC)"
    @go test -tags=e2e -v ./tests/e2e/...

lint: ## Run linter
    @echo "$(CYAN)Running linter...$(NC)"
    @golangci-lint run

docker-build: ## Build Docker image
    @echo "$(CYAN)Building Docker image...$(NC)"
    @docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

docker-up: ## Start development environment
    @echo "$(CYAN)Starting development environment...$(NC)"
    @docker-compose up -d
    @echo "$(GREEN)Development environment started!$(NC)"
    @echo "$(GREEN)PostgreSQL: localhost:5432$(NC)"
    @echo "$(GREEN)Redis: localhost:6379$(NC)"
    @echo "$(GREEN)Prometheus: http://localhost:9090$(NC)"
    @echo "$(GREEN)Grafana: http://localhost:3000 (admin/admin123)$(NC)"
    @echo "$(GREEN)Jaeger: http://localhost:16686$(NC)"

docker-down: ## Stop development environment
    @echo "$(CYAN)Stopping development environment...$(NC)"
    @docker-compose down

docker-logs: ## Show logs from development environment
    @docker-compose logs -f

clean: ## Clean build artifacts
    @echo "$(CYAN)Cleaning build artifacts...$(NC)"
    @rm -rf bin/
    @rm -f coverage.out coverage.html
    @go clean

init: ## Initialize project (run this first)
    @echo "$(CYAN)Initializing ShoPogoda project...$(NC)"
    @cp .env.example .env
    @echo "$(GREEN)Created .env file$(NC)"
    @echo "$(GREEN)Please update .env with your configuration$(NC)"
    @$(MAKE) docker-up
    @echo "$(GREEN)ShoPogoda project initialized! Update your .env file and run 'make build'$(NC)"

migrate: ## Run database migrations
    @echo "$(CYAN)Running database migrations...$(NC)"
    @go run scripts/migrate.go

dev: docker-up build ## Start development environment and build
    @echo "$(GREEN)ShoPogoda development environment ready!$(NC)"

stop: docker-down ## Stop all services
    @echo "$(GREEN)All services stopped$(NC)"

deploy-staging: docker-build ## Deploy to staging
    @echo "$(CYAN)Deploying to staging...$(NC)"
    @docker tag $(DOCKER_IMAGE):latest $(DOCKER_IMAGE):staging
    @echo "$(GREEN)Deployed to staging$(NC)"

deploy-prod: docker-build ## Deploy to production
    @echo "$(CYAN)Deploying to production...$(NC)"
    @docker tag $(DOCKER_IMAGE):latest $(DOCKER_IMAGE):$(VERSION)
    @echo "$(GREEN)Deployed to production$(NC)"
