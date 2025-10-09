# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ShoPogoda (Що Погода - "What Weather" in Ukrainian) is a production-ready Telegram bot for enterprise weather monitoring, environmental alerts, and safety compliance. Built with Go, gotgbot v2, PostgreSQL, Redis, and comprehensive monitoring stack.

**Production Status:**
- ✅ Deployed on Railway (https://shopogoda-svc-production.up.railway.app)
- Database: Supabase PostgreSQL (free tier, 500MB)
- Cache: Upstash Redis (free tier, 10K commands/day)
- Cost: $0/month on free tiers
- Version: 0.1.1 (bugfixes after deploy)

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
    Export       *ExportService
}
```

Services are initialized in `services.New()` with proper dependency chain: DB → Redis → Config → Logger.

**Note**: `LocationService` has been removed - location management is now handled directly by `UserService` with embedded location fields.

### Configuration System

Uses Viper with hierarchical precedence:
1. Environment variables (prefixed with `WB_`)
2. Config files (config.yaml) - **Disabled in production** (Railway deployment)
3. Defaults

Environment variable mapping: `WB_BOT_TOKEN` → `bot.token` in config struct.

**Production Configuration Notes:**
- Railway deployment uses environment variables exclusively
- YAML config loading is disabled to avoid parsing errors
- Supabase requires `PreferSimpleProtocol: true` to disable prepared statement cache
- Upstash Redis auto-enables TLS for non-localhost connections

### Database Models

**Simplified User-Centric Architecture**: Each user has a single embedded location.

Key models with GORM relationships:
- `User` (1:many) → `Subscription`, `AlertConfig`, `WeatherData`
- `User` contains embedded location fields: `location_name`, `latitude`, `longitude`, `country`, `city`
- int64 for User (Telegram user ID), UUIDs for Weather/Alert entities

**Location and Timezone Separation**:
- Location and timezone are completely independent entities
- Location operations (set/clear) do not modify timezone settings
- Timezone operations do not modify location settings
- All timestamps stored in UTC in the database
- User timezone defaults to 'UTC' when not explicitly set
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
- **Notifications**: Dual-platform delivery (Slack + Telegram) with robust error handling
- **Scheduled Notifications**: Timezone-aware daily/weekly weather updates with user preferences
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

### Location and Timezone Separation

**Complete Independence**:
- Location and timezone are separate entities with no cross-dependencies
- Setting/changing location does NOT reset or modify timezone
- Setting/changing timezone does NOT affect location
- Users can have location without timezone, timezone without location, or both independently

**Time Storage**:
- All database timestamps stored in UTC
- User timezone defaults to 'UTC' when not explicitly set (not based on location)
- Time conversion handled on-demand via service layer

**Service Methods**:
```go
// UserService methods for location handling
SetUserLocation(ctx, userID, name, country, city, lat, lon) error    // Does NOT modify timezone
ClearUserLocation(ctx, userID) error                                 // Does NOT modify timezone
GetUserLocation(ctx, userID) (string, float64, float64, error)

// UserService methods for timezone handling
GetUserTimezone(ctx, userID) string                                  // Independent of location status
ConvertToUserTime(ctx, userID, utcTime) time.Time
ConvertToUTC(ctx, userID, localTime) time.Time

// Handler methods for timezone setting
setUserTimezone(bot, ctx, timezone) error                            // Does NOT modify location
```

**Fixed Issues**:
- Removed automatic timezone reset to UTC when setting location
- Removed timezone dependency on location status in `GetUserTimezone`
- Eliminated location checks in timezone operations

### Notification System Implementation

**Comprehensive Notification Management**:
- Full notification preferences UI in bot settings
- Support for multiple notification types: Daily, Weekly, Alerts, Extreme Weather
- Timezone-aware scheduling respects user's local time preferences
- Dual-platform delivery: Telegram (primary) + Slack (enterprise)

**Robust Error Handling**:
- Platform-independent error tracking
- Partial failure tolerance (success if one platform succeeds)
- Detailed logging for notification delivery status
- No complete failure unless both platforms fail

**User Experience**:
- Intuitive UI with add/manage/toggle/delete operations
- UUID-based subscription tracking for security
- Real-time subscription status display
- Seamless integration with Settings menu

**Technical Architecture**:
```go
// NotificationService handles dual-platform delivery
type NotificationService struct {
    config *config.IntegrationsConfig
    logger *zerolog.Logger
    client *http.Client
    bot    *gotgbot.Bot  // Telegram bot instance injection
}

// Key methods for notification delivery
SendTelegramAlert(alert *models.EnvironmentalAlert, user *models.User) error
SendTelegramWeatherUpdate(weather *WeatherData, user *models.User) error
SendSlackAlert(alert *models.EnvironmentalAlert, user *models.User) error
SendSlackWeatherUpdate(weather *WeatherData, subscribers []models.User) error
```

**Scheduler Integration**:
- `SchedulerService` handles timezone-aware notification timing
- Separate processing for alerts (every 10 minutes) and scheduled notifications (hourly check)
- User timezone conversion for accurate local time delivery
- Efficient batching and error handling

### Benefits

- **Reduced Complexity**: 40% fewer database tables and relationships
- **Better Performance**: Eliminates location-related joins
- **Clearer UX**: Single `/setlocation` command vs multiple location commands
- **UTC Consistency**: All times stored uniformly, converted on display
- **Simplified Logic**: User-centric model easier to reason about
- **Independent Settings**: Location and timezone operate independently without side effects

### Data Export System

The bot provides comprehensive data export functionality for compliance, backup, and data portability:

**Export Service Architecture** (`internal/services/export_service.go`):
```go
type ExportService struct {
    db     *gorm.DB
    logger *zerolog.Logger
}

type ExportFormat string
const (
    ExportFormatJSON ExportFormat = "json"
    ExportFormatCSV  ExportFormat = "csv"
    ExportFormatTXT  ExportFormat = "txt"
)

type ExportType string
const (
    ExportTypeWeatherData    ExportType = "weather"
    ExportTypeAlerts         ExportType = "alerts"
    ExportTypeSubscriptions  ExportType = "subscriptions"
    ExportTypeAll            ExportType = "all"
)
```

**Export Data Coverage**:
- **Weather Data**: Last 30 days of weather records (temperature, humidity, pressure, wind, AQI, pollutants)
- **Alerts**: Alert configurations + triggered alerts history (last 90 days)
- **Subscriptions**: Notification preferences, schedules, and settings
- **All Data**: Complete user profile + all above data types

**Export Formats**:
- **JSON**: Machine-readable format with complete data structure for technical use/backup
- **CSV**: Spreadsheet-compatible with separate sections for each data type
- **TXT**: Human-readable format with formatted output for review/reporting

**UI Navigation Flow**:
```
/settings → 📊 Data Export → Choose Data Type → Choose Format → File Delivered
```

**Implementation Features**:
- Temporary file management for secure file transfer
- Comprehensive error handling with user feedback
- Progress indicators during export processing
- Descriptive filenames: `shopogoda_datatype_username_date.ext`
- Professional callback-driven UI with inline keyboards
- Export logging for audit trails

**Security & Performance**:
- Data filtered by user ownership (no cross-user data leakage)
- Reasonable limits: 1000 weather records, 90-day alert history
- Temporary files auto-cleaned after delivery
- Export process isolated from main bot operations

## Testing Approach

### Test Types
- **Unit Tests**: `*_test.go` files alongside source
- **Integration Tests**: `tests/integration/` with testcontainers (PostgreSQL + Redis)
- **Bot Mock Tests**: Handler tests using `tests/helpers/bot_mock.go` infrastructure
- **E2E Tests**: `tests/e2e/` with real bot instance (planned)

### Test Coverage
- **Current**: 30.5% overall coverage
- **Services**: 75.6% (core business logic)
- **Handlers**: 4.2% (bot command handlers)
- **Target**: 40% short-term, 80% long-term

### Bot Mocking Infrastructure

**Location**: `tests/helpers/bot_mock.go`

Reusable infrastructure for testing Telegram bot handler functions without requiring a real bot instance.

**MockBot**: Creates minimal `gotgbot.Bot` instances
```go
mockBot := helpers.NewMockBot()
```

**MockContext**: Flexible `ext.Context` creation with 12 configurable fields
```go
// Simple context
mockCtx := helpers.NewSimpleMockContext(userID, messageText)

// Context with arguments
mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
    Args: []string{"/weather", "New", "York"},
})

// Context with location
mockCtx := helpers.NewMockContextWithLocation(userID, lat, lon)

// Context with callback
mockCtx := helpers.NewMockContextWithCallback(userID, callbackID, data)
```

**Key Features**:
- Context.Args() compatibility via synchronized Update.Message and EffectiveMessage
- Builder pattern with MockContextOptions for flexible configuration
- Support for messages, locations, callbacks, and custom user data

**Usage Example**:
```go
func TestParseLocationFromArgs(t *testing.T) {
    handler := &CommandHandler{}

    mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
        Args: []string{"/weather", "London"},
    })

    result := handler.parseLocationFromArgs(mockCtx.Context)
    assert.Equal(t, "London", result)
}
```

### Test Database
Integration tests use testcontainers for isolated PostgreSQL and Redis instances.

### Running Tests
```bash
make test              # Run all unit tests
make test-coverage     # Generate HTML coverage report
make test-integration  # Run integration tests (requires Docker)
```

See [Testing Guide](docs/TESTING.md) for comprehensive testing documentation.

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

### Production Deployment (Railway)

**Primary deployment platform: Railway + Supabase + Upstash**

```bash
# Deploy to Railway
railway login
railway init
railway up

# Configure environment in Railway dashboard
railway variables set TELEGRAM_BOT_TOKEN=your_token
railway variables set OPENWEATHER_API_KEY=your_key
# ... (see docs/DEPLOYMENT_RAILWAY.md for complete list)
```

**Production Fixes Applied:**
1. YAML config disabled (environment variables only)
2. Supabase compatibility: `PreferSimpleProtocol: true` in GORM
3. AutoMigrate disabled (manual migration script used)
4. Upstash Redis: Automatic TLS for non-localhost hosts

**Live Production:**
- Health: https://shopogoda-svc-production.up.railway.app/health
- Webhook: https://shopogoda-svc-production.up.railway.app/webhook
- Dashboard: https://railway.app/project/191564b9-7a3a-4c8f-bff1-5b214398e3a5

### Docker Production
```bash
make docker-build    # Creates production image (for Fly.io, custom deployment)
```

### Alternative Platforms

**Documented deployment guides:**
- Railway (primary) - `docs/DEPLOYMENT_RAILWAY.md`
- Vercel (serverless) - `docs/DEPLOYMENT_VERCEL.md`
- Fly.io (containers) - `docs/DEPLOYMENT_FLYIO.md`
- Replit (IDE) - `docs/DEPLOYMENT_REPLIT.md`
- GCP - `docs/DEPLOYMENT_GCP.md`

### Environment Variables
All configuration via environment variables. See `.env.example` for full reference.

**Production Mode:**
- Webhook mode (not polling) - required for Railway/Vercel
- Set `BOT_WEBHOOK_MODE=true` and `BOT_WEBHOOK_URL=https://your-domain.com`

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

**Local Development:** <200ms for weather queries through intelligent caching

**Production (Railway):** <500ms average (including cold starts)
- Cold start: ~2-3 seconds (Railway free tier)
- Warm requests: 200-400ms
- Database queries: 100-200ms (Supabase pooler)
- Redis operations: <50ms (Upstash)

### Database Optimization
- Indexes on frequently queried columns (user_id, timestamp)
- Connection pooling (25 connections default, adjusted for Supabase)
- Query optimization for large datasets
- Simplified schema with embedded user locations reduces join complexity
- **Supabase specific:** PreferSimpleProtocol enabled for connection pooler compatibility

### Memory Management
- Bounded cache sizes in Redis (Upstash 10K commands/day limit)
- Graceful degradation on API failures
- Resource limits in containerized deployments (Railway: 1GB RAM on free tier)

### Free Tier Resource Limits

**Railway:**
- 500 instance hours/month (~20 days always-on)
- 1GB RAM
- Webhook mode prevents sleep (always-on required)

**Supabase:**
- 500MB database storage
- 2GB bandwidth/month
- Automatic pause after 7 days inactivity (can be disabled)

**Upstash:**
- 10,000 Redis commands/day (~6.9 commands/minute)
- Increase cache TTL to reduce operations
- Monitor daily usage in dashboard