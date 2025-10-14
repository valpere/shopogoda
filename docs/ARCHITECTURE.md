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
│  PostgreSQL  │  │    Redis     │  │  External    │
│   Database   │  │    Cache     │  │    APIs      │
└──────────────┘  └──────────────┘  └──────────────┘
```

### Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Bot Framework** | gotgbot v2 | Telegram Bot API integration |
| **HTTP Server** | gin-gonic/gin | Webhook endpoint, health checks |
| **Database** | PostgreSQL 15 | Persistent data storage |
| **ORM** | GORM | Database abstraction |
| **Cache** | Redis 7 | Performance optimization |
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
   - Distributed rate limiting via Redis
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
│   Database   │    │    Redis     │    │   Config     │
│  Connection  │    │  Connection  │    │    Viper     │
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
func New(db *gorm.DB, redis *redis.Client, cfg *config.Config, logger *zerolog.Logger) *Services {
    // Initialize core services first
    user := NewUserService(db, redis, logger)
    weather := NewWeatherService(cfg, redis, logger)
    localization := NewLocalizationService(cfg, logger)

    // Initialize dependent services
    alert := NewAlertService(db, user, weather, logger)
    notification := NewNotificationService(cfg, logger, bot)
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
| **UserService** | User management, locations, timezones | DB, Redis |
| **WeatherService** | Weather data retrieval, geocoding | Config, Redis, OpenWeatherMap API |
| **AlertService** | Custom alert configurations | DB, UserService, WeatherService |
| **SubscriptionService** | Notification subscriptions | DB |
| **NotificationService** | Dual-platform delivery | Config, Telegram Bot, Slack/Teams APIs |
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

### Redis Cache Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Redis Cache Layer                        │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  Weather Cache (10min TTL)                                   │
│  ├── Key: weather:lat:lon                                    │
│  └── Value: JSON weather data                                │
│                                                               │
│  Forecast Cache (1hour TTL)                                  │
│  ├── Key: forecast:lat:lon                                   │
│  └── Value: JSON forecast data                               │
│                                                               │
│  Air Quality Cache (10min TTL)                               │
│  ├── Key: air:lat:lon                                        │
│  └── Value: JSON AQI data                                    │
│                                                               │
│  Geocoding Cache (24hour TTL)                                │
│  ├── Key: geocode:location_name                              │
│  └── Value: JSON coordinates                                 │
│                                                               │
│  Rate Limiting (Rolling window)                              │
│  ├── Key: rate:user_id                                       │
│  └── Value: Request counter                                  │
│                                                               │
│  Statistics (24hour rolling)                                 │
│  ├── Key: stats:messages_24h                                 │
│  ├── Key: stats:weather_requests_24h                         │
│  └── Value: Counter with TTL                                 │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### Cache Key Patterns

```go
// Weather data
fmt.Sprintf("weather:%f:%f", lat, lon)

// Forecast data
fmt.Sprintf("forecast:%f:%f", lat, lon)

// Air quality
fmt.Sprintf("air:%f:%f", lat, lon)

// Geocoding
fmt.Sprintf("geocode:%s", strings.ToLower(locationName))

// Rate limiting
fmt.Sprintf("rate:%d", userID)

// Statistics
"stats:messages_24h"
"stats:weather_requests_24h"
```

### Cache Invalidation

- **Time-based**: Automatic expiration via TTL
- **Event-based**: Location changes invalidate related caches
- **Manual**: Admin commands can clear specific caches

### Cache Performance

| Metric | Target | Actual (Production) |
|--------|--------|---------------------|
| Cache Hit Rate | >85% | >85% |
| Cache Read Latency | <50ms | <50ms |
| Cache Write Latency | <100ms | <100ms |

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

### Production Stack (Railway + Supabase + Upstash)

```
┌──────────────────────────────────────────────────────────┐
│                      Internet                            │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────────────┐
│               Telegram API Servers                       │
└────────────────────┬─────────────────────────────────────┘
                     │ HTTPS Webhook
                     ▼
┌──────────────────────────────────────────────────────────┐
│         Railway (Bot Application)                        │
│  ┌────────────────────────────────────────────────────┐  │
│  │    ShoPogoda Bot                                   │  │
│  │    - Webhook endpoint (:8080/webhook)             │  │
│  │    - Health check (:8080/health)                  │  │
│  │    - Metrics (:8080/metrics)                      │  │
│  └────────────────────────────────────────────────────┘  │
└────────────────────┬────────────┬─────────────────────────┘
                     │            │
         ┌───────────┘            └──────────────┐
         ▼                                       ▼
┌─────────────────────┐                ┌─────────────────────┐
│  Supabase           │                │  Upstash Redis      │
│  (PostgreSQL 15)    │                │  (Redis 7)          │
│  - 500MB storage    │                │  - 10K cmds/day     │
│  - Connection pool  │                │  - TLS enabled      │
│  - RLS enabled      │                │  - Cache layer      │
└─────────────────────┘                └─────────────────────┘
```

### Deployment Configuration

**Environment Variables** (all deployments):
```bash
# Core
TELEGRAM_BOT_TOKEN=<from @BotFather>
OPENWEATHER_API_KEY=<from openweathermap.org>

# Mode
BOT_WEBHOOK_MODE=true  # Production: webhook; Development: polling
BOT_WEBHOOK_URL=https://your-domain.com
BOT_WEBHOOK_PORT=8080

# Database (Supabase pooler)
DB_HOST=aws-1-us-east-2.pooler.supabase.com
DB_PORT=6543  # Connection pooler port
DB_NAME=postgres
DB_USER=postgres.<project-ref>
DB_PASSWORD=<supabase-password>
DB_SSL_MODE=require

# Redis (Upstash with TLS)
REDIS_HOST=<region>-<name>.upstash.io
REDIS_PORT=6379  # TLS auto-enabled for non-localhost
REDIS_PASSWORD=<upstash-password>

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Scaling Considerations

**Current Limits (Free Tier)**:
- Railway: 500 execution hours/month (~20 days continuous)
- Supabase: 500MB storage, 2GB bandwidth/month
- Upstash: 10,000 Redis commands/day

**Scaling Options**:
1. **Vertical Scaling**: Upgrade to paid tiers
2. **Horizontal Scaling**: Multiple bot instances with load balancer
3. **Database Scaling**: Read replicas, connection pooling
4. **Cache Scaling**: Redis Cluster

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
- Sensitive data encrypted at rest (DB level)
- TLS for all external communications
- Environment variables for secrets

### Row Level Security (RLS)

Supabase RLS policies ensure data isolation:

```sql
-- Users can only read their own data
CREATE POLICY user_read_own ON users
    FOR SELECT
    USING (telegram_id = current_user_id());

-- Users can only modify their own weather data
CREATE POLICY user_write_own_weather ON weather_data
    FOR ALL
    USING (user_id = current_user_id());
```

See [DATABASE_SECURITY.md](DATABASE_SECURITY.md) for complete RLS documentation.

---

## Scalability Considerations

### Performance Targets

| Metric | Target | Current (Production) |
|--------|--------|---------------------|
| Response Time | <200ms | <500ms (avg) |
| Throughput | 100 req/s | ~10 req/s (free tier) |
| Concurrent Users | 1000+ | <100 (current) |
| Uptime | 99.9% | 99.5%+ |

### Bottlenecks & Mitigation

**1. Database Connections**
- **Issue**: Limited connection pool
- **Solution**: Connection pooling with Supabase pooler
- **Configuration**: 25 max connections

**2. Redis Command Limits**
- **Issue**: Upstash 10K commands/day
- **Solution**: Optimize cache TTLs, batch operations
- **Monitor**: Daily command usage

**3. Railway Execution Hours**
- **Issue**: 500 hours/month on free tier
- **Solution**: Upgrade to paid tier or optimize wake/sleep patterns

### Future Scaling Architecture

```
┌────────────────────────────────────────────────────────┐
│            Load Balancer (NGINX/HAProxy)               │
└───────────┬────────────────────────┬───────────────────┘
            │                        │
    ┌───────▼───────┐        ┌───────▼───────┐
    │  Bot Instance │        │  Bot Instance │
    │      #1       │        │      #2       │
    └───────┬───────┘        └───────┬───────┘
            │                        │
            └───────────┬────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│  PostgreSQL  │ │Redis Cluster │ │  External    │
│   Primary    │ │ (3 nodes)    │ │    APIs      │
│   + Replica  │ │              │ │              │
└──────────────┘ └──────────────┘ └──────────────┘
```

---

## Monitoring & Observability

### Metrics Collection

**Prometheus Metrics** (`pkg/metrics/`):
- Request counter by command
- Response time histogram
- Error rate by type
- Cache hit/miss ratio
- Database connection pool stats
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
- Redis connectivity check
- External API availability
- Queue depth monitoring

---

## Additional Resources

- **[API Reference](API_REFERENCE.md)**: Complete service layer API
- **[Database Migration Guide](DATABASE_MIGRATION_GUIDE.md)**: Schema management
- **[Database Security](DATABASE_SECURITY.md)**: RLS and security practices
- **[Deployment Guide](DEPLOYMENT.md)**: Production deployment instructions
- **[Testing Guide](TESTING.md)**: Testing strategies and best practices

---

**Last Updated**: 2025-10-14
**Version**: 0.1.2-dev
**Status**: Production Deployed
