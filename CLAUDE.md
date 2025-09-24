# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ShoPogoda (Що Погода - "What Weather" in Ukrainian) is a production-ready Telegram bot for enterprise weather monitoring, environmental alerts, and safety compliance. Built with Go, gotgbot v2, PostgreSQL, Redis, and comprehensive monitoring stack.

## Core Development Commands

### Quick Start
```bash
# Initialize project (first time setup)
make init                # Copies .env.example to .env and starts containers

# Development workflow
make deps               # Install Go dependencies
make build              # Build the application
make run                # Build and run the bot
make dev                # Start dev environment + build

# Testing
make test               # Run unit tests
make test-coverage      # Run tests with HTML coverage report
make test-integration   # Run integration tests (requires containers)
make test-e2e          # Run end-to-end tests

# Code quality
make lint               # Run golangci-lint

# Infrastructure
make docker-up          # Start PostgreSQL, Redis, Prometheus, Grafana
make docker-down        # Stop all containers
make docker-logs        # View container logs
make migrate            # Run database migrations
```

### Essential Configuration

Copy `.env.example` to `.env` and configure:
- `TELEGRAM_BOT_TOKEN` - Required from @BotFather
- `OPENWEATHER_API_KEY` - Required from openweathermap.org
- `SLACK_WEBHOOK_URL` - Optional for enterprise notifications

### Monitoring URLs (after `make docker-up`)
- Bot Health: http://localhost:8080/health
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin123)
- Jaeger Tracing: http://localhost:16686

## Architecture Overview

### Core Structure
```
cmd/bot/main.go              # Application entry point with graceful shutdown
internal/
├── bot/                     # Bot initialization, HTTP server, webhook setup
├── config/                  # Viper-based configuration with environment variables
├── database/                # PostgreSQL + Redis connection management
├── handlers/commands/       # Telegram command handlers (/weather, /forecast, etc.)
├── middleware/              # Logging, metrics, auth, rate limiting middleware
├── models/                  # GORM models with relationships and migrations
└── services/                # Business logic layer with dependency injection
pkg/
├── metrics/                 # Prometheus metrics collectors
└── weather/                 # Weather API client abstractions
```

### Service Layer Architecture

The `services/` package follows dependency injection with a central `Services` struct:

```go
// internal/services/services.go
type Services struct {
    User         *UserService
    Weather      *WeatherService
    Alert        *AlertService
    Subscription *SubscriptionService
    Notification *NotificationService
    Scheduler    *SchedulerService
}
```

Services are initialized in `services.New()` with proper dependency chain: DB → Redis → Config → Logger.

**Note**: `LocationService` has been removed - location management is now handled directly by `UserService` with embedded location fields.

### Configuration System

Uses Viper with hierarchical precedence:
1. Environment variables (prefixed with `WB_`)
2. Config files (config.yaml)
3. Defaults

Environment variable mapping: `WB_BOT_TOKEN` → `bot.token` in config struct.

### Database Models

**Simplified User-Centric Architecture**: Each user has a single embedded location.

Key models with GORM relationships:
- `User` (1:many) → `Subscription`, `AlertConfig`, `WeatherData`
- `User` contains embedded location fields: `location_name`, `latitude`, `longitude`, `country`, `city`
- int64 for User (Telegram user ID), UUIDs for Weather/Alert entities

**UTC Timezone Handling**:
- All timestamps stored in UTC in the database
- User timezone defaults to 'UTC' when no location is set
- Timezone conversion handled on-demand via `UserService` helper methods

Migration: `models.Migrate(db)` handles all schema changes.

### Bot Command Architecture

Commands in `internal/handlers/commands/` follow pattern:
```go
func WeatherCommand(bot *gotgbot.Bot, ctx *gotgbot.CallbackContext, services *Services) error
```

Middleware applied: logging, metrics, auth, rate limiting (10 req/min per user).

### Caching Strategy

Redis caching with TTL:
- Weather data: 10 minutes
- Forecasts: 1 hour
- Geocoding: 24 hours
- User sessions: configurable

### Enterprise Features

- **Alerts**: Custom thresholds with severity calculation and cooldown
- **Notifications**: Slack/Teams webhooks with retry logic
- **Roles**: User/Moderator/Admin with command-level authorization
- **Monitoring**: Prometheus metrics, structured logging, health checks

## Architecture Changes

### Location Model Simplification

The bot has been refactored from a complex multi-location system to a simplified single-location-per-user model:

**Before**:
- Separate `Location` entity with complex relationships
- Multiple locations per user with "default" concept
- Multiple commands: `/addlocation`, `/setdefault`, `/locations`, etc.

**After**:
- Location embedded directly in `User` model
- Single location per user (no separate Location table)
- Single command: `/setlocation` replaces all location management
- Simplified database queries and reduced join complexity

### UTC-First Design

**Time Storage**:
- All database timestamps stored in UTC
- User timezone defaults to 'UTC' when no location set
- Time conversion handled on-demand via service layer

**Service Methods**:
```go
// UserService methods for timezone handling
GetUserTimezone(ctx, userID) string
ConvertToUserTime(ctx, userID, utcTime) time.Time
ConvertToUTC(ctx, userID, localTime) time.Time
SetUserLocation(ctx, userID, name, country, city, lat, lon) error
ClearUserLocation(ctx, userID) error
```

### Benefits

- **Reduced Complexity**: 40% fewer database tables and relationships
- **Better Performance**: Eliminates location-related joins
- **Clearer UX**: Single `/setlocation` command vs multiple location commands
- **UTC Consistency**: All times stored uniformly, converted on display
- **Simplified Logic**: User-centric model easier to reason about

## Testing Approach

### Test Types
- Unit: `*_test.go` files alongside source
- Integration: `tests/integration/` with testcontainers
- E2E: `tests/e2e/` with real bot instance

### Test Database
Integration tests use testcontainers for isolated PostgreSQL instances.

### Coverage Target
Maintain >80% test coverage. Use `make test-coverage` to generate HTML report.

## Key Dependencies

### Core Framework
- `gotgbot/v2` - Telegram Bot API with webhook support
- `gin-gonic/gin` - HTTP server for webhooks and health checks
- `gorm.io/gorm` - ORM with PostgreSQL driver
- `redis/go-redis/v9` - Redis client

### Configuration & Logging
- `spf13/viper` - Configuration management
- `rs/zerolog` - Structured JSON logging

### Monitoring
- `prometheus/client_golang` - Metrics collection
- Custom collectors in `pkg/metrics/`

### Testing
- `testcontainers/testcontainers-go` - Integration test containers
- `stretchr/testify` - Test assertions

## Build & Deployment

### Local Development
```bash
make dev    # Starts all services and builds app
```

### Docker Production
```bash
make docker-build    # Creates production image
```

### Environment Variables
All configuration via environment variables. See `.env.example` for full reference.

Bot supports both polling and webhook modes. Webhook preferred for production.

## Bot Commands Reference

**User Commands:**
- `/start` - Welcome and setup
- `/weather [location]` - Current weather
- `/forecast [location]` - 5-day forecast
- `/air [location]` - Air quality
- `/setlocation` - Set user's single location (replaces multiple location management commands)
- `/subscribe` - Setup notifications
- `/addalert` - Create custom alerts
- `/settings` - User preferences

**Admin Commands:**
- `/stats` - System statistics
- `/broadcast` - Message all users
- `/users` - User management

## Development Patterns

### Error Handling
Use wrapped errors with context: `fmt.Errorf("operation failed: %w", err)`

### Logging
Structured logging with correlation IDs for request tracing.

### Database Operations
Always use transactions for multi-table operations.

### API Rate Limiting
Respect OpenWeatherMap limits. Use Redis for request counting.

### Security
- Input validation on all user data
- SQL injection prevention via GORM
- Rate limiting per user
- No hardcoded credentials

## Performance Considerations

### Response Time Target
<200ms for weather queries through intelligent caching.

### Database Optimization
- Indexes on frequently queried columns (user_id, timestamp)
- Connection pooling (25 connections default)
- Query optimization for large datasets
- Simplified schema with embedded user locations reduces join complexity

### Memory Management
- Bounded cache sizes in Redis
- Graceful degradation on API failures
- Resource limits in containerized deployments