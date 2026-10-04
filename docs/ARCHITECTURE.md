# ShoPogoda Architecture

Comprehensive system architecture documentation for the ShoPogoda weather bot.

## Table of Contents

- [System Overview](#system-overview)
- [Architecture Layers](#architecture-layers)
- [Service Layer Design](#service-layer-design)
- [Database Schema](#database-schema)
- [Caching Strategy](#caching-strategy)
- [External Integrations](#external-integrations)
- [Deployment Architecture](#deployment-architecture)
- [Security Architecture](#security-architecture)
- [Scalability Considerations](#scalability-considerations)

---

## System Overview

ShoPogoda is built with a modern, layered architecture optimized for maintainability, testability, and production deployment.

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                         Telegram API                          │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                      Bot Layer (Webhook/Polling)              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   HTTP       │  │   Telegram   │  │   Health     │       │
│  │   Server     │  │   Handler    │  │   Checks     │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                    Middleware Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Logging    │  │   Metrics    │  │  Rate        │       │
│  │              │  │              │  │  Limiting    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                     Handler Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Command    │  │   Callback   │  │    Admin     │       │
│  │   Handlers   │  │   Handlers   │  │   Handlers   │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                     Service Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │    User      │  │   Weather    │  │    Alert     │       │
│  │   Service    │  │   Service    │  │   Service    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ Subscription │  │Notification  │  │  Scheduler   │       │
│  │   Service    │  │   Service    │  │   Service    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Export     │  │Localization  │  │    Demo      │       │
│  │   Service    │  │   Service    │  │   Service    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└────────────────────────┬─────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│    SQLite    │  │  In-process  │  │  External    │
│  (one file)  │  │    Cache     │  │    APIs      │
└──────────────┘  └──────────────┘  └──────────────┘
```

### Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Bot Framework** | gotgbot v2 | Telegram Bot API integration |
| **HTTP Server** | gin-gonic/gin | Webhook endpoint, health checks |
| **Database** | SQLite (`glebarez/sqlite`, pure Go) | Persistent data storage (single file) |
| **ORM** | GORM | Database abstraction |
| **Cache** | In-process (`internal/cache`) | TTL + bounded LRU cache |
| **Logging** | zerolog | Structured JSON logging |
| **Metrics** | Prometheus | Monitoring and metrics |
| **Configuration** | Viper | Hierarchical configuration |
| **Language** | Go 1.24+ | Core application language |

---

## Architecture Layers

### 1. Bot Layer (`internal/bot/`)

**Responsibility**: Telegram Bot API integration and webhook management

**Key Components**:
- **Bot Initialization**: Creates and configures bot instance
- **HTTP Server**: Gin server for webhook endpoint
- **Dispatcher**: Routes updates to appropriate handlers
- **Webhook Setup**: Configures webhook with Telegram API
- **Health Checks**: Exposes `/health` and `/metrics` endpoints

**Entry Point**: `cmd/bot/main.go`

```go
// Bot initialization pattern
bot, err := gotgbot.NewBot(token)
dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
    MaxRoutines: 20,
})

// Register handlers
dispatcher.AddHandler(handlers.WeatherCommand(...))
```

### 2. Middleware Layer (`internal/middleware/`)

**Responsibility**: Cross-cutting concerns

**Middleware Components**:

1. **Logging Middleware**
   - Request/response logging
   - Correlation IDs for tracing
   - Structured log output

2. **Metrics Middleware**
   - Request counter
   - Response time histogram
   - Error rate tracking

3. **Authentication Middleware**
   - User registration/verification
   - Session management
   - Role-based authorization

4. **Rate Limiting Middleware**
   - Per-user rate limits (10 req/min)
   - In-process rate limiting (`golang.org/x/time/rate`)
   - Graceful cleanup of expired limiters

### 3. Handler Layer (`internal/handlers/`)

**Responsibility**: Process Telegram updates and coordinate service calls

**Handler Types**:

1. **Command Handlers** (`handlers/commands/`)
   - User commands: `/weather`, `/forecast`, `/air`
   - Settings commands: `/setlocation`, `/subscribe`, `/settings`
   - Admin commands: `/stats`, `/broadcast`, `/users`

2. **Callback Handlers** (`handlers/callbacks/`)
   - Settings callbacks (language, timezone, units)
   - Notification management callbacks
   - Data export callbacks

**Handler Pattern**:
```go
func WeatherCommand(
    bot *gotgbot.Bot,
    ctx *ext.Context,
    services *services.Services,
) error {
    // 1. Extract user and parameters
    // 2. Call service layer
    // 3. Format response
    // 4. Send message
}
```

### 4. Service Layer (`internal/services/`)

**Responsibility**: Business logic and data operations

See [Service Layer Design](#service-layer-design) for detailed documentation.

### 5. Data Layer

**Components**:
- **Models** (`internal/models/`): GORM data models
- **Database** (`internal/database/`): Connection management
- **Migrations**: Schema versioning

---

## Service Layer Design

### Services Overview

The service layer encapsulates all business logic with clear separation of concerns.

```go
// Central services struct with dependency injection
type Services struct {
    User         *UserService
    Weather      *WeatherService
    Alert        *AlertService
    Subscription *SubscriptionService
    Notification *NotificationService
    Scheduler    *SchedulerService
    Export       *ExportService
    Localization *LocalizationService
    Demo         *DemoService
}
```

### Service Dependencies

```
┌───────────────────────────────────────────────────────────┐
│                   Service Initialization                   │
└───────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Database   │    │ In-process   │    │   Config     │
│  (SQLite)    │    │    Cache     │    │    Viper     │
└──────┬───────┘    └──────┬───────┘    └──────┬───────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
         ┌─────────────────┴─────────────────┐
         ▼                                   ▼
┌──────────────────┐              ┌──────────────────┐
│  Core Services   │              │ Dependent        │
│  - UserService   │───────────── │   Services       │
│  - WeatherService│              │ - AlertService   │
│  - Localization  │              │ - Subscription   │
└──────────────────┘              │ - Notification   │
                                  │ - Scheduler      │
                                  └──────────────────┘
```

### Service Initialization Pattern

```go
// services/services.go
func New(db *gorm.DB, cfg *config.Config, logger *zerolog.Logger, metricsCollector *metrics.Metrics) *Services {
    // Initialize core services first
    user := NewUserService(db, metricsCollector, logger, startTime)
    weather := NewWeatherService(&cfg.Weather, logger)
    localization := NewLocalizationService(logger)

    // Initialize dependent services
    alert := NewAlertService(db)
    notification := NewNotificationService(&cfg.Integrations, logger)
    // ... other services

    return &Services{
        User:         user,
        Weather:      weather,
        Alert:        alert,
        // ... other services
    }
}
```

### Key Service Responsibilities

| Service | Responsibility | Dependencies |
|---------|---------------|--------------|
| **UserService** | User management, locations, timezones | DB, in-process cache |
| **WeatherService** | Weather data retrieval, geocoding | Config, in-process cache, OpenWeatherMap API |
| **AlertService** | Custom alert configurations | DB |
| **SubscriptionService** | Notification subscriptions | DB |
| **NotificationService** | Dual-platform delivery with in-memory retry queue (`delivery_queue.go`) | Config, Telegram Bot, Slack/Teams APIs |
| **SchedulerService** | Background job scheduling | All services |
| **ExportService** | Data export (JSON/CSV/TXT) | DB |
| **LocalizationService** | Multi-language translation | Config |
| **DemoService** | Demo data management | DB |

For complete API documentation, see [API_REFERENCE.md](API_REFERENCE.md).

---

## Database Schema

### Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                          users                              │
├─────────────────────────────────────────────────────────────┤
│ telegram_id (PK)                                            │
│ username, first_name, last_name                             │
│ language_code, timezone                                     │
│ location_name, country, city, latitude, longitude           │
│ role (user/moderator/admin)                                 │
│ is_active, created_at, updated_at                           │
└───────────────┬─────────────────────────────────────────────┘
                │
      ┌─────────┼─────────┬─────────────┬────────────────┐
      │         │         │             │                │
      ▼         ▼         ▼             ▼                ▼
┌───────────┐ ┌────────────┐ ┌──────────────┐ ┌─────────────────┐
│  weather  │ │   alert    │ │subscription  │ │ environmental   │
│   _data   │ │  _configs  │ │      s       │ │    _alerts      │
├───────────┤ ├────────────┤ ├──────────────┤ ├─────────────────┤
│ id (PK)   │ │ id (PK)    │ │ id (PK)      │ │ id (PK)         │
│ user_id(FK│ │ user_id(FK)│ │ user_id (FK) │ │ alert_config_id │
│ temp, hum │ │ alert_type │ │ type         │ │   (FK)          │
│ pressure  │ │ threshold  │ │ frequency    │ │ user_id (FK)    │
│ wind_*    │ │ condition  │ │ delivery_time│ │ severity        │
│ aqi, *_co │ │ is_active  │ │ is_active    │ │ message         │
│ timestamp │ │ created_at │ │ created_at   │ │ created_at      │
└───────────┘ └────────────┘ └──────────────┘ └─────────────────┘
```

### Key Models

#### User Model
```go
type User struct {
    TelegramID   int64  `gorm:"primaryKey"`
    Username     string
    FirstName    string
    LastName     string
    LanguageCode string  // 'en', 'uk', 'de', 'fr', 'es'
    Timezone     string  // 'Europe/Kyiv', 'America/New_York', etc.

    // Embedded location (single location per user)
    LocationName string
    Country      string
    City         string
    Latitude     float64
    Longitude    float64

    Role      string    // 'user', 'moderator', 'admin'
    IsActive  bool
    CreatedAt time.Time
    UpdatedAt time.Time

    // Relationships
    WeatherData         []WeatherData         `gorm:"foreignKey:UserID"`
    AlertConfigs        []AlertConfig         `gorm:"foreignKey:UserID"`
    Subscriptions       []Subscription        `gorm:"foreignKey:UserID"`
    EnvironmentalAlerts []EnvironmentalAlert  `gorm:"foreignKey:UserID"`
}
```

#### WeatherData Model
```go
type WeatherData struct {
    ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
    UserID      int64     `gorm:"index"`
    Temperature float64
    Humidity    int
    Pressure    int
    WindSpeed   float64
    WindDeg     int
    Description string

    // Air Quality (optional)
    AQI   int
    PM25  float64
    PM10  float64
    CO    float64
    NO2   float64
    O3    float64
    SO2   float64

    Timestamp time.Time `gorm:"index"`
    CreatedAt time.Time
}
```

### Database Indexes

**Optimized for common queries:**

```sql
-- User lookups
CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_active_role ON users(is_active, role);

-- Weather data queries
CREATE INDEX idx_weather_data_user_timestamp ON weather_data(user_id, timestamp DESC);

-- Alert lookups
CREATE INDEX idx_alert_configs_user_active ON alert_configs(user_id, is_active);

-- Subscription queries
CREATE INDEX idx_subscriptions_user_active ON subscriptions(user_id, is_active);
CREATE INDEX idx_subscriptions_active_type ON subscriptions(is_active, subscription_type);
```

---

## Caching Strategy

### In-Process Cache Architecture

`internal/cache` is an in-process cache: TTL per entry, bounded LRU (weather cache 2000 entries, user cache 10000). It is lost on restart. `WeatherService.SetCache` is a test seam.

```
┌──────────────────────────────────────────────────────────────┐
│                  In-Process Cache (internal/cache)            │
├──────────────────────────────────────────────────────────────┤
│  Current weather   (10min TTL)   weather:current:lat:lon      │
│  Forecast          (1hour TTL)   weather:forecast:lat:lon:N   │
│  Air quality       (30min TTL)   weather:air:lat:lon          │
│  Geocoding         (24hour TTL)  geocode:<name>               │
│  Reverse geocoding (24hour TTL)  reverse_geocode:lat:lon      │
│  User profile      (1hour TTL)   user:<id> (invalidated on    │
│                                  updates)                     │
└──────────────────────────────────────────────────────────────┘
```

### Cache Key Patterns

```go
fmt.Sprintf("weather:current:%.4f:%.4f", lat, lon)
fmt.Sprintf("weather:forecast:%.4f:%.4f:%d", lat, lon, days)
fmt.Sprintf("weather:air:%.4f:%.4f", lat, lon)
fmt.Sprintf("geocode:%s", normalizedName)
fmt.Sprintf("reverse_geocode:%.4f:%.4f", lat, lon)
fmt.Sprintf("user:%d", userID)
```

### Rate Limiting and Activity Counters

- Rate limiting is in-process (`golang.org/x/time/rate`), 10 req/min per user
- Activity counters (messages, weather requests) are in-memory atomics, reset on restart and labelled "since start" (`MessagesSinceStart`, `WeatherRequestsSinceStart`)
- "New users (24h)" is a real 24-hour database query

### Cache Invalidation

- **Time-based**: Automatic expiration via TTL
- **Event-based**: User profile (`user:<id>`) is invalidated on updates

### Cache Performance

Hit rate and latency depend on traffic and host; measure on your deployment (cache hits are served in-process without network round trips).

---

## External Integrations

### OpenWeatherMap API

**Integration**: `pkg/weather/openweather_client.go`

```go
type OpenWeatherClient struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

// APIs used:
// - Current Weather: /data/2.5/weather
// - 5-day Forecast: /data/2.5/forecast
// - Air Quality: /data/2.5/air_pollution
// - Geocoding: /geo/1.0/direct
```

**Rate Limits**:
- Free tier: 60 calls/minute, 1,000,000 calls/month
- Caching reduces actual API calls by >85%

### Slack Integration

**Integration**: `internal/services/notification_service.go`

```go
func (s *NotificationService) SendSlackAlert(
    alert *models.EnvironmentalAlert,
    user *models.User,
) error {
    // Formats alert as Slack Block Kit message
    // Posts to webhook URL from configuration
}
```

**Features**:
- Rich formatting with Block Kit
- Severity-based color coding
- Actionable links

### Microsoft Teams Integration

**Similar pattern to Slack** with Adaptive Cards formatting.

---

## Deployment Architecture

### Single-Instance Stack

```
┌──────────────────────────────────────────────────────────┐
│                      Internet                            │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────────────┐
│               Telegram API Servers                       │
└────────────────────┬─────────────────────────────────────┘
                     │ HTTPS Webhook (or polling)
                     ▼
┌──────────────────────────────────────────────────────────┐
│     Single binary / one container (exactly 1 instance)   │
│  ┌────────────────────────────────────────────────────┐  │
│  │    ShoPogoda Bot                                   │  │
│  │    - Webhook endpoint (:8080/webhook)              │  │
│  │    - Health check (:8080/health)                   │  │
│  │    - Metrics (:8080/metrics)                       │  │
│  │    - In-process cache, rate limiter, retry queue   │  │
│  └──────────────────────┬─────────────────────────────┘  │
└─────────────────────────┼────────────────────────────────┘
                          ▼
              ┌───────────────────────┐
              │  Volume: /app/data    │
              │  shopogoda.db (SQLite)│
              └───────────────────────┘
```

SQLite runs in WAL mode with `busy_timeout` 5000, `foreign_keys` on and `SetMaxOpenConns(1)`. A single writer means the bot must run as exactly one instance/replica (Kubernetes: `replicas: 1`, strategy `Recreate`, PVC).

### Deployment Configuration

**Environment Variables** (all deployments):
```bash
# Core
TELEGRAM_BOT_TOKEN=<from @BotFather>
OPENWEATHER_API_KEY=<from openweathermap.org>

# Mode
BOT_WEBHOOK_MODE=true  # Webhook needs a public HTTPS URL; polling also works
BOT_WEBHOOK_URL=https://your-domain.com
BOT_WEBHOOK_PORT=8080

# Database (SQLite file)
DB_PATH=/app/data/shopogoda.db  # default ./data/shopogoda.db

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

See [Deployment Guide](DEPLOYMENT.md) for Docker Compose, Kubernetes and bare-binary setups. Backups: `sqlite3 <file> ".backup out.db"` or Litestream.

### Scaling Considerations

The design is intentionally single-instance: state is one SQLite file and the cache, rate limiter, counters and retry queue are per-process.

**Scaling Options**:
1. **Vertical Scaling**: Give the single instance more CPU/RAM
2. **Cache sizing**: Adjust the bounded LRU sizes in `internal/cache` usage
3. **Beyond one instance**: Would require replacing SQLite with a networked database and the in-process cache, rate limiter and queue with shared equivalents (not implemented)

---

## Security Architecture

### Authentication & Authorization

```go
// Role-based access control
type Role string
const (
    RoleUser      Role = "user"
    RoleModerator Role = "moderator"
    RoleAdmin     Role = "admin"
)

// Authorization middleware
func RequireRole(minRole Role) ext.HandlerFunc {
    return func(b *gotgbot.Bot, ctx *ext.Context) error {
        user := getUserFromContext(ctx)
        if !hasRole(user, minRole) {
            return ErrUnauthorized
        }
        return nil
    }
}
```

### Input Validation

- All user inputs sanitized
- SQL injection prevention via GORM
- XSS prevention in message formatting
- Rate limiting per user (10 req/min)

### Data Protection

- Passwords/tokens never logged
- Protect the SQLite file with filesystem permissions and volume/disk encryption as needed
- TLS for all external communications
- Environment variables for secrets

### Data Isolation

All queries are scoped to the owning user (no cross-user data leakage); the bot is the only client of the SQLite file, so there is no database-level access layer to secure.

---

## Scalability Considerations

### Performance Targets

| Metric | Target |
|--------|--------|
| Response Time | <200ms (cache hits) |

Measure throughput, latency and uptime on your own host; no benchmarks are claimed here.

### Bottlenecks & Mitigation

**1. Single SQLite writer**
- **Issue**: One connection (`SetMaxOpenConns(1)`) serializes DB access
- **Mitigation**: WAL mode, `busy_timeout` 5000, short transactions

**2. Per-process state**
- **Issue**: Cache, counters and delivery queue are lost on restart
- **Mitigation**: TTLs are short; counters are labelled "since start"; state worth keeping lives in SQLite

**3. OpenWeatherMap quota**
- **Mitigation**: In-process caching (see TTLs above)

---

## Monitoring & Observability

### Metrics Collection

**Prometheus Metrics** (`pkg/metrics/`):
- Request counter by command
- Response time histogram
- Error rate by type
- Cache hit/miss ratio
- Database connection stats
- Active users gauge

### Structured Logging

**Zerolog** with JSON output:
```json
{
  "level": "info",
  "time": "2025-10-14T10:30:00Z",
  "user_id": 123456789,
  "command": "/weather",
  "duration_ms": 150,
  "cache_hit": true,
  "message": "Weather command executed successfully"
}
```

### Health Checks

**Endpoints**:
- `/health`: Basic health status
- `/metrics`: Prometheus metrics

**Future Enhancements**:
- Database connectivity check
- External API availability
- Queue depth monitoring

---

## Additional Resources

- **[API Reference](API_REFERENCE.md)**: Complete service layer API
- **[Deployment Guide](DEPLOYMENT.md)**: Production deployment instructions
- **[Testing Guide](TESTING.md)**: Testing strategies and best practices

---

**Last Updated**: 2025-10-14
**Version**: 0.1.2-dev
**Status**: Production Deployed (single instance)
