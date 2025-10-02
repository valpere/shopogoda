# Changelog

All notable changes to ShoPogoda will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### In Progress
- Test coverage improvements (27.4% → 28.4%, target: 40%)
- Video walkthrough

### Added
- **Alert Service Integration Tests**: Comprehensive integration tests for alert management
  - 5 test functions with 10 subtests covering CRUD operations and alert triggering
  - TestIntegration_AlertServiceCreateAlert: Single and multiple alert type creation
  - TestIntegration_AlertServiceGetUserAlerts: Active alert filtering and non-existent user handling
  - TestIntegration_AlertServiceUpdateAlert: Alert threshold updates
  - TestIntegration_AlertServiceDeleteAlert: Soft delete verification
  - TestIntegration_AlertServiceCheckAlerts: Alert triggering, cooldown periods, inactive alerts
  - Alert service method coverage: CreateAlert: 100%, GetUserAlerts: 100%, CheckAlerts: 62.9%, DeleteAlert: 100%, UpdateAlert: 85.7%
  - Integration tests add 6.0% coverage to internal packages
- **Demo Service Integration Tests**: Comprehensive testcontainers-based integration tests
  - 6 test functions with 9 subtests covering seed, clear, reset, and data validation
  - PostgreSQL 15 Alpine + Redis 7 Alpine containers for realistic testing
  - Demo service function coverage: 0% → 63-90% (SeedDemoData: 63.6%, createDemoWeatherData: 90.5%)
  - Integration tests cover 11.1% of services package
  - Weather data validation: temperature ranges, humidity, pressure, temporal variation
  - Alert and subscription configuration verification
- **Timezone Validation Tests**: Added comprehensive unit tests for timezone validation
  - 10 test cases covering valid timezones (UTC, America/New_York, Europe/Kyiv, etc.)
  - Invalid timezone detection (malformed, empty, partial timezone names)
  - Edge case handling (empty string defaults to Local timezone)

### Fixed
- **Demo Service ClearDemoData Bug**: Fixed SQL error when deleting demo user
  - Root cause: User table uses `id` as primary key, related tables use `user_id` foreign key
  - Separated deletion logic: related tables use `WHERE user_id = ?`, User table uses `WHERE id = ?`
  - Fixed 2 failing integration tests (clear data + idempotency)
- **Service Test SQL Expectations**: Fixed 18 failing test cases caused by GORM query pattern changes
  - Alert service: LIMIT parameter expectations
  - Export service: User query ORDER BY and LIMIT clauses
  - User service: Timezone and location test SQL patterns
  - Services package coverage: 71.3% → 75.6% (+4.3%)
  - Overall coverage: 27.4% → 28.5% (+1.1%)

---

## [0.1.0-demo] - 2025-01-02

### Major Features

#### Core Weather Services
- Real-time weather data via OpenWeatherMap API
- 5-day weather forecasts with 3-hour intervals
- Air quality monitoring (AQI and pollutant tracking)
- Location management with GPS and text-based input
- Timezone-aware weather displays

#### Enterprise Features
- **Custom Alert System**: User-defined thresholds for temperature, humidity, AQI
  - Severity calculation (Low, Medium, High, Critical)
  - Alert cooldown periods to prevent spam
  - Alert history tracking
- **Scheduled Notifications**: Timezone-aware daily/weekly weather updates
  - Dual-platform delivery (Telegram + Slack)
  - User-configurable notification times
  - Robust error handling with partial failure tolerance
- **Role-Based Access Control**: Admin, Moderator, User roles with command-level authorization
- **Monitoring Stack**: Prometheus metrics, Grafana dashboards, Jaeger tracing

#### Multi-Language Support
- Complete localization in 5 languages: English (en), Ukrainian (uk), German (de), French (fr), Spanish (es)
- Dynamic language switching via `/settings`
- Persistent language preferences per user
- Automatic fallback to English for missing translations

#### Data Management
- **Data Export System**: Export user data in multiple formats
  - Weather data (last 30 days)
  - Alert history (last 90 days)
  - Subscriptions and preferences
  - Formats: JSON, CSV, TXT
- **Demo Mode**: Comprehensive demonstration system
  - Auto-seeds realistic demo data on startup (when DEMO_MODE=true)
  - Demo user (ID: 999999999) with Kyiv, Ukraine location
  - 24 hours of weather data with natural temperature patterns
  - 3 alert configurations, 3 notification subscriptions
  - Admin commands: `/demoreset`, `/democlear`
  - Perfect for testing, presentations, and development
- **Location Simplification**: Embedded location model (single location per user)
- **Timezone Independence**: Location and timezone managed separately

#### Infrastructure
- PostgreSQL database with GORM ORM
- Redis caching with TTL (10-min weather, 1-hour forecasts, 24-hour geocoding)
- Docker containerization for all services
- CI/CD pipeline with automated testing and deployment
- Health checks and metrics endpoints
- Graceful shutdown handling

### Bot Commands

#### User Commands
- `/start` - Welcome message and bot introduction
- `/help` - Display all available commands
- `/weather [location]` - Get current weather
- `/forecast [location]` - Get 5-day forecast
- `/air [location]` - Get air quality information
- `/setlocation` - Set user's location (GPS or text)
- `/subscribe` - Manage notification subscriptions
- `/addalert` - Create custom weather alerts
- `/settings` - User preferences (language, timezone, data export, notifications)

#### Admin Commands
- `/stats` - System statistics and metrics
- `/broadcast` - Send message to all users
- `/users` - User management
- `/demoreset` - Clear and re-seed demo data
- `/democlear` - Remove all demo data

### Technical Improvements

#### Testing
- Unit tests for core services (version, demo, commands, services)
- Integration tests with testcontainers
- Test coverage: 27.4%
- Mock testing for external APIs
- Comprehensive demo service tests with edge case validation

#### Code Quality
- golangci-lint integration
- gofmt code formatting
- Pre-commit hooks for code validation
- Comprehensive error handling

#### CI/CD Pipeline
- Automated testing on pull requests
- Docker image building and pushing
- Code coverage reporting (Codecov)
- Security scanning (GitHub CodeQL)
- Deployment automation for main/develop branches

### Dependencies

#### Core Dependencies
- Go 1.24.6
- gotgbot/v2 v2.0.0-rc.25 (Telegram Bot API)
- gorm.io/gorm v1.25.6 (ORM)
- redis/go-redis/v9 v9.0.0 (Redis client)
- gin-gonic/gin v1.9.1 (HTTP server)
- spf13/viper v1.18.2 (Configuration)
- rs/zerolog v1.31.0 (Structured logging)

#### Monitoring
- prometheus/client_golang v1.23.2
- uber/jaeger-client-go v2.30.0

#### Testing
- testcontainers-go v0.39.0
- stretchr/testify v1.9.0

#### GitHub Actions
- actions/checkout v4
- actions/setup-go v5
- docker/build-push-action v6
- codecov/codecov-action v5
- actions/github-script v8
- golangci/golangci-lint-action v8

### Architecture Changes

#### Location Model Simplification
**Before**: Separate `Location` entity with complex relationships, multiple locations per user
**After**: Location embedded in `User` model, single location per user

**Benefits**:
- 40% fewer database tables
- Eliminated complex join queries
- Clearer user experience (single `/setlocation` command)
- Better performance

#### Timezone Independence
- Location and timezone completely decoupled
- Setting location doesn't reset timezone
- All timestamps stored in UTC
- On-demand conversion via service layer

#### Notification System
- Comprehensive notification UI in bot settings
- Multiple notification types with granular control
- Timezone-aware scheduling
- Dual-platform delivery with error tolerance

### Known Issues

#### Test Failures (Deferred to v0.2.0)
- GORM v1.31.0 upgrade blocked due to breaking SQL query changes
- golangci-lint v8 upgrade requires code adjustments
- gotgbot v2.0.0-rc.33 has breaking changes

#### Limitations
- Free-tier OpenWeatherMap limits (60 calls/min)
- Single location per user (by design)
- No historical weather data

---

## [0.2.0] - Production Beta (Planned: Q2 2025)

### Planned Features

#### Testing & Quality
- Increase test coverage to 60%+
- E2E test suite with real Telegram API
- Performance benchmarks
- Load testing (1000+ req/min)
- Security audit

#### Advanced Weather Features
- Historical weather data (past 7 days)
- Weather comparisons (current vs. historical)
- Severe weather warnings
- Hourly forecasts (48 hours)
- Weather radar and satellite imagery

#### User Experience
- Interactive weather maps
- Customizable notification templates
- Weather widgets for group chats
- Inline query support
- Voice message weather reports

#### Enterprise Enhancements
- Multi-user organization support
- Team dashboards
- Admin analytics dashboard
- Custom webhook endpoints
- SLA-based rate limiting

#### Infrastructure
- Horizontal scaling support
- PostgreSQL replication
- Redis Cluster
- Automated backup and restore
- Blue-green deployments

#### Dependency Upgrades
- Upgrade GORM to v1.31.0 (with test fixes)
- Upgrade golangci-lint to v8 (with code quality improvements)
- Upgrade gotgbot to stable v2.0.0

---

## [1.0.0] - Stable Release (Planned: Q3 2025)

### Planned Features

#### Premium Features
- Subscription tiers (Free, Pro, Enterprise)
- Payment integration
- Extended forecasts (15 days)
- Unlimited custom alerts

#### Advanced Alerts
- AI-powered alert recommendations
- Complex conditions (AND/OR logic)
- Alert templates library
- Geofencing-based alerts

#### Integration Ecosystem
- Zapier integration
- IFTTT support
- Discord integration
- Microsoft Teams native app
- REST API for third-party apps

#### Mobile & Web
- Progressive Web App (PWA)
- Mobile-optimized dashboard
- QR code onboarding

---

## [2.0.0] - AI & Automation (Planned: Q4 2025)

### Planned Features

#### AI-Powered Features
- Natural language queries ("Will I need an umbrella?")
- Smart alert suggestions
- Weather-based activity recommendations
- Conversational bot mode

#### Automation
- Auto-tuning alerts
- Smart notification scheduling
- Predictive infrastructure scaling

#### Developer Experience
- Plugin system for extensions
- Developer API with SDKs
- GraphQL API
- Real-time event streaming

---

## Release Management

### B. Release Management Improvements

Planned improvements for systematic release process:

#### Version Control
- Semantic versioning enforcement
- Automated version bumping
- Git tag automation
- Version constants in code

#### Changelog Automation
- Conventional commits enforcement
- Auto-generated changelogs from commits
- Release notes generation
- Breaking change detection

#### Release Workflow
- Automated GitHub releases
- Docker image tagging (version-based)
- Multi-environment deployments
- Rollback procedures

#### Quality Gates
- Minimum test coverage requirements
- Linting pass required
- Security scan pass required
- Performance benchmarks

---

## Contributing

See our [contribution guidelines](CONTRIBUTING.md) for details on:
- Code style and standards
- Pull request process
- Testing requirements
- Documentation standards

---

## Version History Summary

| Version | Release Date | Status | Highlights |
|---------|--------------|--------|------------|
| 0.1.0-demo | 2025-01-02 | 🚧 In Progress | Initial demo release |
| 0.2.0 | Q2 2025 | 📋 Planned | Production beta |
| 1.0.0 | Q3 2025 | 📋 Planned | Stable release |
| 2.0.0 | Q4 2025 | 💭 Concept | AI & Automation |

---

**Last Updated**: 2025-01-02
**Maintained by**: [@valpere](https://github.com/valpere)
