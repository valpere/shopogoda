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

.PHONY: help build run test clean docker-build docker-up docker-down deps fmt lint typecheck check init migrate dev stop deploy-staging deploy-prod

help: ## Show this help message
	@echo "$(CYAN)ShoPogoda (Що Погода) - Development Commands$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install Go dependencies
	@echo "$(CYAN)Installing dependencies...$(NC)"
	@go mod download
	@go mod tidy

build: deps ## Build the application
	@echo "$(CYAN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p bin
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

fmt: ## Format Go code
	@echo "$(CYAN)Formatting Go code...$(NC)"
	@gofmt -s -w .
	@goimports -w .

lint: ## Run linter
	@echo "$(CYAN)Running linter...$(NC)"
	@golangci-lint run --timeout=5m

typecheck: ## Run type check
	@echo "$(CYAN)Running type check...$(NC)"
	@go build -o /dev/null ./...

check: fmt lint typecheck test ## Run all checks (format, lint, typecheck, test)
	@echo "$(GREEN)All checks passed!$(NC)"

clean: ## Clean build artifacts
	@echo "$(CYAN)Cleaning build artifacts...$(NC)"
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean

docker-build: ## Build Docker image
	@echo "$(CYAN)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

docker-up: ## Start development environment
	@echo "$(CYAN)Starting development environment...$(NC)"
	@docker compose -f docker/docker-compose.yml up -d
	@echo "$(GREEN)Development environment started!$(NC)"
	@echo "$(GREEN)PostgreSQL: localhost:5432$(NC)"
	@echo "$(GREEN)Redis: localhost:6380$(NC)"
	@echo "$(GREEN)Prometheus: http://localhost:9090$(NC)"
	@echo "$(GREEN)Grafana: http://localhost:3000 (admin/admin123)$(NC)"
	@echo "$(GREEN)Jaeger: http://localhost:16686$(NC)"

docker-down: ## Stop development environment
	@echo "$(CYAN)Stopping development environment...$(NC)"
	@docker compose -f docker/docker-compose.yml down

docker-logs: ## Show logs from development environment
	@docker compose -f docker/docker-compose.yml logs -f

docker-clean: ## Clean Docker containers and images
	@echo "$(CYAN)Cleaning Docker resources...$(NC)"
	@docker compose -f docker/docker-compose.yml down -v --remove-orphans
	@docker system prune -f

init: ## Initialize project (run this first)
	@echo "$(CYAN)Initializing ShoPogoda project...$(NC)"
	@cp .env.example .env || echo "$(RED).env.example not found$(NC)"
	@echo "$(GREEN)Created .env file (if it didn't exist)$(NC)"
	@echo "$(GREEN)Please update .env with your configuration:$(NC)"
	@echo "$(GREEN)  - TELEGRAM_BOT_TOKEN$(NC)"
	@echo "$(GREEN)  - OPENWEATHER_API_KEY$(NC)"
	@$(MAKE) docker-up
	@echo "$(GREEN)ShoPogoda project initialized!$(NC)"

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

# Development helpers
env-check: ## Check if required environment variables are set
	@echo "$(CYAN)Checking environment variables...$(NC)"
	@test -n "$(TELEGRAM_BOT_TOKEN)" || echo "$(RED)TELEGRAM_BOT_TOKEN not set$(NC)"
	@test -n "$(OPENWEATHER_API_KEY)" || echo "$(RED)OPENWEATHER_API_KEY not set$(NC)"
	@echo "$(GREEN)Environment check complete$(NC)"

logs: ## Show application logs
	@echo "$(CYAN)Showing logs...$(NC)"
	@tail -f logs/shopogoda.log 2>/dev/null || echo "$(RED)No log file found$(NC)"

status: ## Show service status
	@echo "$(CYAN)Service Status:$(NC)"
	@docker compose -f docker/docker-compose.yml ps

# Quick commands
quick-build: ## Quick build without deps check
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/bot/main.go

quick-test: ## Quick test without verbose output
	@go test ./...

# Installation helpers
install-tools: ## Install development tools
	@echo "$(CYAN)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(GREEN)Development tools installed$(NC)"

# Database helpers
db-reset: ## Reset database (WARNING: destroys all data)
	@echo "$(RED)WARNING: This will destroy all database data!$(NC)"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ]
	@docker compose -f docker/docker-compose.yml down -v
	@docker compose -f docker/docker-compose.yml up -d postgres redis
	@sleep 5
	@$(MAKE) migrate
	@echo "$(GREEN)Database reset complete$(NC)"

# Show usage
usage: help