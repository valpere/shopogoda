# Changelog

All notable changes to ShoPogoda will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **Admin & Statistics Enhancement**: Real metrics collection and activity tracking
  - **Real Metrics Integration**: Replaced placeholder values with actual Prometheus metrics
    - Cache hit rate: Real-time extraction from Prometheus gauges with label matching
    - Average response time: Histogram-based calculation from handler duration sum/count
    - System uptime: Dynamic calculation showing service availability over 24-hour period
  - **Activity Tracking**: Redis-based counters with 24-hour rolling windows
    - Message counter: Tracks total bot messages sent to users
    - Weather request counter: Tracks API calls for weather, forecast, and air quality
    - Automatic TTL management: 24-hour expiry for rolling statistics
  - **Non-Blocking Metrics**: Warning-level logging for metric failures to prevent request failures
  - **Test-Friendly Metrics**: Graceful handling of duplicate Prometheus registrations in test environment
  - **Implementation Details**:
    - `UserService.IncrementMessageCounter()`: Redis INCR with TTL check
    - `UserService.IncrementWeatherRequestCounter()`: Redis INCR with TTL check
    - `Metrics.GetCacheHitRate()`: Real Prometheus gauge extraction using DTO and label matching
    - `Metrics.GetAverageResponseTime()`: Real histogram sum/count calculation (seconds to milliseconds)
    - Message tracking added to: Start command
    - Weather tracking added to: CurrentWeather, Forecast, AirQuality commands
  - **Services Architecture**: Metrics collector passed through dependency injection
    - `Services.New()` signature updated to accept `*metrics.Metrics`
    - `UserService` now tracks application `startTime` for uptime calculation
    - All test files updated to pass metrics collector instances
  - **Test Coverage**: 11 new tests for metric extraction (GetCacheHitRate, GetAverageResponseTime)

### In Progress
- Test coverage improvements (27.4% → 29.9%, target: 30%)
- Video walkthrough

---

## [0.1.1] - 2025-01-10

### Fixed
- **Location Dialog Consistency**: Standardized location setup UI across all commands
  - `/setlocation` command: Replaced single "Share Location" button with 3-button dialog
  - `CurrentWeather` command: Applied same 3-button dialog when location needed
  - All location dialogs now show: "Set Location by Name", "Set Location by Coordinates", "Back to Start"
  - Removed deprecated Share Location functionality entirely
  - Added `handleBackCallback` for "Back to Start" navigation
  - All buttons support multi-language localization
- **New User Auto-Registration**: Fixed silent failures for first-time users
  - New users who start with commands other than `/start` are now auto-registered
  - Friendly welcome message shown even when first command isn't `/start`
  - Eliminated "record not found" errors in logs for new users
  - Added `ensureUserRegistered()` helper function
- **Deployment Workflow False Alarms**: Fixed GitHub Actions deployment false failures
  - Added conditional checks to release workflow to prevent false alarms
  - Jobs now only run on version tag pushes, not regular commits
  - Proper dependency handling with `always()` condition

### Changed
- Version updated from 0.1.0-dev to 0.1.1
- Improved consistency across all location-related user interactions

---

### Added
- **Bot Mocking Infrastructure**: Reusable test infrastructure for Telegram bot handler testing
  - MockBot: Minimal gotgbot.Bot instances with configurable test data
  - MockContext: Flexible ext.Context creation with MockContextOptions builder pattern
  - 12 configurable fields: UserID, Username, FirstName, LastName, ChatID, MessageID, MessageText, Args, CallbackID, Data, Latitude, Longitude
  - Helper functions: NewMockBot(), NewMockContext(), NewSimpleMockContext(), NewMockContextWithLocation(), NewMockContextWithCallback()
  - Context.Args() compatibility: Synchronized Update.Message and EffectiveMessage for proper argument parsing
  - Comprehensive mock infrastructure tests: 100% coverage of bot_mock.go (TestNewMockBot, TestNewMockContext, TestMockContextDefaults)
  - First practical usage: TestParseLocationFromArgs with 7 test cases validating location parsing from command arguments
  - tests/helpers package coverage: 0% → 24.5% (+24.5%)
  - Handler package coverage: 4.0% → 4.2% (+0.2%)
  - Overall coverage: 29.4% → 29.9% (+0.5%)
  - Infrastructure enables future testing of all handler command functions (Start, Help, Weather, Forecast, Settings, etc.)
- **User Service Integration Tests**: Comprehensive integration tests for user management
  - 7 test functions with 16 subtests covering user lifecycle, settings, location, timezone, caching
  - TestIntegration_UserServiceRegisterUser: New user creation and upsert behavior
  - TestIntegration_UserServiceGetUser: Database retrieval and Redis caching validation
  - TestIntegration_UserServiceUpdateUserSettings: Language, timezone, units updates
  - TestIntegration_UserServiceLocationManagement: Set, get, clear location operations
  - TestIntegration_UserServiceTimezoneManagement: Timezone defaults, conversions (UTC ↔ local)
  - TestIntegration_UserServiceGetActiveUsers: Active user filtering
  - TestIntegration_UserServiceCacheInvalidation: Cache lifecycle after updates
  - User service method coverage: RegisterUser: 100%, GetUser: 100%, UpdateUserSettings: 100%, Location methods: 100%
- **Weather Service Integration Tests**: Comprehensive integration tests for weather data and caching
  - 7 test functions with 13 subtests covering weather retrieval, geocoding, air quality, and caching
  - TestIntegration_WeatherServiceGetCurrentWeather: Current weather API and cache validation
  - TestIntegration_WeatherServiceGetForecast: Forecast data caching behavior
  - TestIntegration_WeatherServiceGetAirQuality: Air quality data caching (30-min TTL)
  - TestIntegration_WeatherServiceGeocodeLocation: Location geocoding with cache normalization
  - TestIntegration_WeatherServiceGetCompleteWeatherData: Combined weather + air quality data, graceful air quality fallback
  - TestIntegration_WeatherServiceGetLocationName: Reverse geocoding with fallback to coordinates
  - TestIntegration_WeatherServiceCacheTTL: Cache TTL verification (10min weather, 1hr forecast, 30min air, 24hr geocode)
  - Caching strategy validation: Redis key format, TTL accuracy, cache hit behavior
  - Error handling: Empty location names, missing air quality data
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
- **Subscription Service Integration Tests**: Comprehensive integration tests for notification subscriptions
  - 7 test functions with 15 subtests covering CRUD operations and subscription management
  - TestIntegration_SubscriptionServiceCreateSubscription: Subscription creation for all types
  - TestIntegration_SubscriptionServiceGetUserSubscriptions: Active subscription filtering
  - TestIntegration_SubscriptionServiceUpdateSubscription: Subscription modification (time, frequency)
  - TestIntegration_SubscriptionServiceToggleSubscription: Enable/disable subscriptions
  - TestIntegration_SubscriptionServiceDeleteSubscription: Soft delete verification (is_active = false)
  - TestIntegration_SubscriptionServiceGetSubscriptionsByType: Type-based filtering (Daily, Weekly, Alerts, Extreme)
  - TestIntegration_SubscriptionServiceCacheInvalidation: Redis cache lifecycle after updates
  - Subscription service method coverage: CreateSubscription: 100%, GetUserSubscriptions: 100%, UpdateSubscription: 100%, DeleteSubscription: 100%
- **Export Service Integration Tests**: Comprehensive integration tests for data export functionality
  - 9 test functions with 12 subtests covering all export formats and data types
  - TestIntegration_ExportServiceExportWeatherDataJSON: Weather data export in JSON format with structure validation
  - TestIntegration_ExportServiceExportSubscriptionsCSV: Subscription export in multi-section CSV format
  - TestIntegration_ExportServiceExportAlertsTXT: Alert configuration and triggered alert export in human-readable TXT
  - TestIntegration_ExportServiceExportAllData: Combined export of all data types (weather + alerts + subscriptions)
  - TestIntegration_ExportServiceEmptyData: Empty data handling (no records)
  - TestIntegration_ExportServiceNonExistentUser: Non-existent user error handling
  - TestIntegration_ExportServiceDateFiltering: Date-based filtering (30-day weather window, 90-day alert history for compliance)
  - TestIntegration_ExportServiceAllFormats: Format validation (JSON parsing, CSV multi-section content, TXT readability)
  - TestIntegration_ExportServiceFilenameFormat: Filename structure verification (shopogoda_{type}_{username}_{date}.{ext})
  - Export service coverage with localization integration: ExportUserData comprehensive testing
- **Timezone Validation Tests**: Added comprehensive unit tests for timezone validation
  - 10 test cases covering valid timezones (UTC, America/New_York, Europe/Kyiv, etc.)
  - Invalid timezone detection (malformed, empty, partial timezone names)
  - Edge case handling (empty string defaults to Local timezone)
- **Handler Helper Function Tests**: Added comprehensive unit tests for handler utility functions
  - 7 test functions with 47 test cases covering helper utilities
  - TestGetAQIDescription: Air quality index description mapping (14 cases: Good, Moderate, Unhealthy, Hazardous)
  - TestGetHealthRecommendation: Health recommendations based on AQI (12 cases)
  - TestGetLocalizedUnitsText: Unit system localization (4 cases: metric, imperial, fallback)
  - TestGetLocalizedRoleName: User role localization (4 cases: Admin, Moderator, User, default)
  - TestGetLocalizedStatusText: Active/inactive status localization (2 cases)
  - TestGetSubscriptionTypeText: Subscription type localization (5 cases: Daily, Weekly, Alerts, Extreme, Unknown)
  - TestGetFrequencyText: Frequency localization (6 cases: Hourly, Every3Hours, Every6Hours, Daily, Weekly, Unknown)
  - Handler package coverage: 2.1% → 4.0% (+1.9%)
  - Overall coverage: 28.6% → 29.4% (+0.8%)

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
