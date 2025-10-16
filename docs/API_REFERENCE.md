# ShoPogoda API Reference

**Version:** 0.1.2-dev
**Last Updated:** 2025-10-11

This document provides comprehensive API documentation for ShoPogoda's service layer. The service layer is the core business logic of the application, providing a clean interface between handlers and data persistence.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Service Initialization](#service-initialization)
- [UserService](#userservice)
- [WeatherService](#weatherservice)
- [AlertService](#alertservice)
- [SubscriptionService](#subscriptionservice)
- [NotificationService](#notificationservice)
- [SchedulerService](#schedulerservice)
- [ExportService](#exportservice)
- [LocalizationService](#localizationservice)
- [DemoService](#demoservice)
- [Error Handling](#error-handling)
- [Caching Strategy](#caching-strategy)
- [Common Patterns](#common-patterns)

---

## Architecture Overview

### Service Layer Pattern

ShoPogoda follows a service-oriented architecture where business logic is encapsulated in service structs. Each service:

- Has a single responsibility
- Depends on abstractions (database, cache, config)
- Uses dependency injection for testability
- Returns domain models
- Handles its own caching and error handling

### Services Container

All services are organized in a central `Services` struct:

```go
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

### Dependency Chain

```plaintext
Database (PostgreSQL) → GORM
Cache (Redis) → go-redis
Config → Viper
Logger → zerolog
Metrics → Prometheus
    ↓
Services (Dependency Injection)
    ↓
Handlers (Telegram Commands)
```

---

## Service Initialization

### Creating the Services Container

```go
import (
    "github.com/valpere/shopogoda/internal/services"
    "github.com/valpere/shopogoda/internal/config"
    "github.com/valpere/shopogoda/pkg/metrics"
)

// Initialize dependencies
cfg := config.Load()
db := database.Connect(cfg.Database)
redis := database.ConnectRedis(cfg.Redis)
logger := zerolog.New(os.Stdout)
metricsCollector := metrics.New()

// Create services container
svcs := services.New(db, redis, cfg, logger, metricsCollector)

// Start background scheduler
ctx := context.Background()
svcs.StartScheduler(ctx)

// Cleanup on shutdown
defer svcs.Stop()
```

**Function Signature:**

```go
func New(
    db *gorm.DB,
    redis *redis.Client,
    cfg *config.Config,
    logger *zerolog.Logger,
    metricsCollector *metrics.Metrics,
) *Services
```

---

## UserService

Manages user accounts, locations, timezones, and system statistics.

### Constructor

```go
func NewUserService(
    db *gorm.DB,
    redis *redis.Client,
    metricsCollector *metrics.Metrics,
    startTime time.Time,
) *UserService
```

### User Management

#### RegisterUser

Registers or updates a user from Telegram data (upsert operation). **Automatically detects and normalizes the user's language** from Telegram's IETF language code.

```go
func (s *UserService) RegisterUser(ctx context.Context, tgUser *gotgbot.User) error
```

**Parameters:**

- `ctx` - Context for cancellation and tracing
- `tgUser` - Telegram user object from gotgbot

**Returns:**

- `error` - Error if registration fails

**Language Detection:**

When a user first interacts with the bot, `RegisterUser` automatically:
1. Extracts the user's `language_code` from Telegram (IETF language tag like "en-US", "uk-UA")
2. Normalizes it to a supported full IETF language code ("en-US", "uk-UA", "de-DE", "fr-FR", "es-ES")
3. Defaults to "en-US" (English) if the language is not supported
4. Stores the normalized language in the user's profile

**Example:**

```go
user := &gotgbot.User{
    Id:           123456789,
    FirstName:    "John",
    LastName:     "Doe",
    Username:     "johndoe",
    LanguageCode: "en-US",  // IETF tag from Telegram
}

// Language automatically normalized: "en-US" → "en"
if err := services.User.RegisterUser(ctx, user); err != nil {
    return fmt.Errorf("failed to register user: %w", err)
}
```

**Language Code Examples:**
- `"en-US"` → `"en-US"` (English - exact match)
- `"en-GB"` → `"en-US"` (English GB maps to US)
- `"uk-UA"` → `"uk-UA"` (Ukrainian - exact match)
- `"de-DE"` → `"de-DE"` (German - exact match)
- `"fr-FR"` → `"fr-FR"` (French - exact match)
- `"fr-CA"` → `"fr-FR"` (French Canada maps to France)
- `"es-ES"` → `"es-ES"` (Spanish - exact match)
- `"es-MX"` → `"es-ES"` (Spanish Mexico maps to Spain)
- `"en"` → `"en-US"` (simple code maps to full tag)
- `"uk"` → `"uk-UA"` (simple code maps to full tag)
- `"it-IT"` → `"en-US"` (Italian not supported, defaults to English)

#### NormalizeLanguageCode

Normalizes a Telegram IETF language tag to a supported full IETF language code. This is called automatically by `RegisterUser` but can be used independently for language detection.

```go
func (s *UserService) NormalizeLanguageCode(telegramLangCode string) string
```

**Parameters:**

- `telegramLangCode` - IETF language tag from Telegram (e.g., "en-US", "uk-UA", "en", "fr-CA")

**Returns:**

- `string` - Normalized full IETF language code ("en-US", "uk-UA", "de-DE", "fr-FR", "es-ES") or "en-US" if unsupported

**Supported Languages:**
- `en-US` - English (United States)
- `uk-UA` - Ukrainian (Ukraine) - Українська
- `de-DE` - German (Germany) - Deutsch
- `fr-FR` - French (France) - Français
- `es-ES` - Spanish (Spain) - Español

**Normalization Rules:**
1. Returns exact match if input is already a supported full IETF tag
2. Extracts primary language code (before hyphen) for regional variants
3. Maps simple codes and regional variants to canonical full IETF tags
4. Converts to lowercase and trims whitespace
5. Returns "en-US" as default for unsupported languages

**Mapping Examples:**
- Exact matches: `"en-US"` → `"en-US"`, `"uk-UA"` → `"uk-UA"`
- Simple codes: `"en"` → `"en-US"`, `"uk"` → `"uk-UA"`, `"de"` → `"de-DE"`
- Regional variants: `"en-GB"` → `"en-US"`, `"fr-CA"` → `"fr-FR"`, `"es-MX"` → `"es-ES"`
- Unsupported: `"it-IT"` → `"en-US"`, `"pt-BR"` → `"en-US"`

**Example:**

```go
// Exact matches (supported full IETF tags)
lang := services.User.NormalizeLanguageCode("en-US")    // "en-US"
lang := services.User.NormalizeLanguageCode("uk-UA")    // "uk-UA"
lang := services.User.NormalizeLanguageCode("de-DE")    // "de-DE"

// Regional variants map to canonical tags
lang := services.User.NormalizeLanguageCode("en-GB")    // "en-US"
lang := services.User.NormalizeLanguageCode("fr-CA")    // "fr-FR"
lang := services.User.NormalizeLanguageCode("es-MX")    // "es-ES"

// Simple codes map to full IETF tags
lang := services.User.NormalizeLanguageCode("en")       // "en-US"
lang := services.User.NormalizeLanguageCode("uk")       // "uk-UA"
lang := services.User.NormalizeLanguageCode("de")       // "de-DE"

// Unsupported languages default to English
lang := services.User.NormalizeLanguageCode("it-IT")    // "en-US"
lang := services.User.NormalizeLanguageCode("pt-BR")    // "en-US"

// Edge cases
lang := services.User.NormalizeLanguageCode("")         // "en-US"
lang := services.User.NormalizeLanguageCode("EN-us")    // "en-us" (normalized to lowercase)
```

#### GetUser

Retrieves user data with Redis caching (1-hour TTL).

```go
func (s *UserService) GetUser(ctx context.Context, userID int64) (*models.User, error)
```

**Parameters:**

- `ctx` - Context for cancellation and tracing
- `userID` - Telegram user ID

**Returns:**

- `*models.User` - User object
- `error` - Error if user not found or retrieval fails

**Cache Key:** `user:{userID}`
**Cache TTL:** 1 hour

**Example:**

```go
user, err := services.User.GetUser(ctx, 123456789)
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return fmt.Errorf("user not found")
    }
    return fmt.Errorf("failed to get user: %w", err)
}

fmt.Printf("User: %s (Role: %s)\n", user.Username, user.Role)
```

#### UpdateUserSettings

Updates user settings and invalidates cache.

```go
func (s *UserService) UpdateUserSettings(
    ctx context.Context,
    userID int64,
    settings map[string]interface{},
) error
```

**Parameters:**

- `ctx` - Context for cancellation and tracing
- `userID` - Telegram user ID
- `settings` - Map of field names to values

**Returns:**

- `error` - Error if update fails

**Example:**

```go
settings := map[string]interface{}{
    "units":          "metric",
    "language":       "en-US",  // Full IETF language tag
    "timezone":       "America/New_York",
    "notifications":  true,
}

if err := services.User.UpdateUserSettings(ctx, userID, settings); err != nil {
    return fmt.Errorf("failed to update settings: %w", err)
}
```

### Location Management

**Important:** Location and timezone are completely independent. Setting location does NOT modify timezone, and vice versa.

#### SetUserLocation

Sets user's location without affecting timezone.

```go
func (s *UserService) SetUserLocation(
    ctx context.Context,
    userID int64,
    locationName string,
    country string,
    city string,
    lat float64,
    lon float64,
) error
```

**Parameters:**

- `locationName` - Full location name
- `country` - Country code (ISO 3166-1 alpha-2)
- `city` - City name
- `lat` - Latitude (-90 to 90)
- `lon` - Longitude (-180 to 180)

**Example:**

```go
err := services.User.SetUserLocation(
    ctx,
    userID,
    "New York, NY, USA",
    "US",
    "New York",
    40.7128,
    -74.0060,
)
```

#### ClearUserLocation

Clears user's location without affecting timezone.

```go
func (s *UserService) ClearUserLocation(ctx context.Context, userID int64) error
```

#### GetUserLocation

Retrieves user's location information.

```go
func (s *UserService) GetUserLocation(
    ctx context.Context,
    userID int64,
) (string, float64, float64, error)
```

**Returns:**

- `string` - Location name
- `float64` - Latitude
- `float64` - Longitude
- `error` - Error if user not found or has no location

### Timezone Management

#### GetUserTimezone

Returns user's timezone, defaulting to "UTC" if not set. Independent of location status.

```go
func (s *UserService) GetUserTimezone(ctx context.Context, userID int64) string
```

**Returns:** Timezone string (e.g., "America/New_York", "UTC")

#### ConvertToUserTime

Converts UTC time to user's local time.

```go
func (s *UserService) ConvertToUserTime(
    ctx context.Context,
    userID int64,
    utcTime time.Time,
) time.Time
```

**Example:**

```go
utcTime := time.Now().UTC()
localTime := services.User.ConvertToUserTime(ctx, userID, utcTime)
fmt.Printf("User's local time: %s\n", localTime.Format("15:04 MST"))
```

#### ConvertToUTC

Converts user's local time to UTC.

```go
func (s *UserService) ConvertToUTC(
    ctx context.Context,
    userID int64,
    localTime time.Time,
) time.Time
```

### Statistics

#### GetSystemStats

Returns comprehensive system statistics including real Prometheus metrics.

```go
func (s *UserService) GetSystemStats(ctx context.Context) (*SystemStats, error)
```

**Returns:**

```go
type SystemStats struct {
    TotalUsers            int64
    ActiveUsers           int64
    TotalSubscriptions    int64
    TotalAlerts           int64
    CacheHitRate          float64  // Real Prometheus gauge
    AverageResponseTime   float64  // Real histogram average
    SystemUptime          string   // Calculated from start time
    Messages24h           int64    // Redis counter
    WeatherRequests24h    int64    // Redis counter
}
```

**Example:**

```go
stats, err := services.User.GetSystemStats(ctx)
if err != nil {
    return fmt.Errorf("failed to get stats: %w", err)
}

fmt.Printf("Users: %d (Active: %d)\n", stats.TotalUsers, stats.ActiveUsers)
fmt.Printf("Cache Hit Rate: %.2f%%\n", stats.CacheHitRate*100)
fmt.Printf("Avg Response Time: %.2fms\n", stats.AverageResponseTime*1000)
fmt.Printf("Uptime: %s\n", stats.SystemUptime)
```

#### GetUserStatistics

Returns user-focused statistics.

```go
func (s *UserService) GetUserStatistics(ctx context.Context) (*UserStatistics, error)
```

**Returns:**

```go
type UserStatistics struct {
    TotalUsers    int64
    ActiveUsers   int64
    UsersByRole   map[string]int64
    RecentSignups int64  // Last 24 hours
}
```

#### GetActiveUsers

Returns all active users.

```go
func (s *UserService) GetActiveUsers(ctx context.Context) ([]models.User, error)
```

### Activity Tracking

#### IncrementMessageCounter

Increments 24-hour rolling message counter in Redis.

```go
func (s *UserService) IncrementMessageCounter(ctx context.Context) error
```

**Redis Key:** `stats:messages_24h`
**TTL:** 24 hours (auto-set on first increment)

#### IncrementWeatherRequestCounter

Increments 24-hour rolling weather request counter in Redis.

```go
func (s *UserService) IncrementWeatherRequestCounter(ctx context.Context) error
```

**Redis Key:** `stats:weather_requests_24h`
**TTL:** 24 hours

### Language Management

#### UpdateUserLanguage

Updates user's language preference.

```go
func (s *UserService) UpdateUserLanguage(
    ctx context.Context,
    userID int64,
    language string,
) error
```

**Supported Languages:** en-US, uk-UA, de-DE, fr-FR, es-ES (full IETF tags)

### Role Management

**Important:** Role management requires Admin privileges. The system enforces role hierarchy and prevents demoting the last admin.

#### ChangeUserRole

Changes a user's role with admin validation and safety checks.

```go
func (s *UserService) ChangeUserRole(
    ctx context.Context,
    adminID int64,
    targetUserID int64,
    newRole models.UserRole,
) error
```

**Parameters:**

- `adminID` - ID of the admin performing the role change
- `targetUserID` - ID of the user whose role is being changed
- `newRole` - New role (RoleUser=1, RoleModerator=2, RoleAdmin=3)

**Returns:**

- `error` - Error if validation fails or operation unsuccessful

**Security & Validation:**

- Verifies admin has `RoleAdmin` permission
- Prevents demoting the last admin in the system
- Invalidates user cache immediately after change
- Logs role changes for audit trail

**Example:**

```go
// Promote user to Moderator
err := services.User.ChangeUserRole(ctx, adminID, targetUserID, models.RoleModerator)
if err != nil {
    if strings.Contains(err.Error(), "insufficient permissions") {
        return fmt.Errorf("only admins can change roles")
    }
    if strings.Contains(err.Error(), "cannot demote the last admin") {
        return fmt.Errorf("system must have at least one admin")
    }
    return fmt.Errorf("role change failed: %w", err)
}

fmt.Printf("User promoted to Moderator\n")
```

**Error Scenarios:**

- `"insufficient permissions"` - Caller is not an admin
- `"cannot demote the last admin"` - Attempting to demote the only remaining admin
- `"user not found"` - Target user doesn't exist
- Database errors - Transaction failures

**Cache Invalidation:**

- Automatically invalidates `user:{targetUserID}` cache key
- Ensures subsequent requests see updated role immediately

#### GetRoleName

Returns human-readable role name for localization.

```go
func (s *UserService) GetRoleName(role models.UserRole) string
```

**Parameters:**

- `role` - User role (RoleUser, RoleModerator, RoleAdmin)

**Returns:**

- `string` - Role name ("User", "Moderator", "Admin")

**Example:**

```go
user, err := services.User.GetUser(ctx, userID)
if err != nil {
    return err
}

roleName := services.User.GetRoleName(user.Role)
fmt.Printf("User role: %s\n", roleName)  // "User role: Admin"
```

**User Roles:**

| Role | Value | Description |
|------|-------|-------------|
| RoleUser | 1 | Default role, basic bot features |
| RoleModerator | 2 | Enhanced permissions, user assistance |
| RoleAdmin | 3 | Full system access, user management |

**Command Permissions:**

- `/promote` - Admin only
- `/demote` - Admin only
- `/stats` - Admin/Moderator
- `/broadcast` - Admin only
- `/users` - Admin/Moderator

**Implementation Notes:**

- Role changes are atomic (transaction-wrapped by GORM)
- Admin count check uses database query to prevent race conditions
- Role enum values are sequential for future role hierarchy checks
- Default role for new users is `RoleUser`

---

## WeatherService

Handles weather data retrieval with caching and geocoding.

### Constructor

```go
func NewWeatherService(
    cfg *config.WeatherConfig,
    redis *redis.Client,
    logger *zerolog.Logger,
) *WeatherService
```

### Weather Retrieval

#### GetCompleteWeatherData

Gets combined weather and air quality data.

```go
func (s *WeatherService) GetCompleteWeatherData(
    ctx context.Context,
    lat float64,
    lon float64,
) (*WeatherData, error)
```

**Returns:**

```go
type WeatherData struct {
    Temperature    float64
    FeelsLike      float64
    TempMin        float64
    TempMax        float64
    Pressure       int
    Humidity       int
    WindSpeed      float64
    WindDeg        int
    Clouds         int
    Description    string
    Icon           string
    Visibility     int
    Rain1h         float64
    Snow1h         float64
    Country        string
    Sunrise        time.Time
    Sunset         time.Time
    Timezone       int
    LocationName   string

    // Air Quality
    AQI            int
    PM25           float64
    PM10           float64
    NO2            float64
    O3             float64
    SO2            float64
    CO             float64
}
```

**Cache:** 10 minutes for weather, 30 minutes for air quality

**Example:**

```go
weather, err := services.Weather.GetCompleteWeatherData(ctx, 40.7128, -74.0060)
if err != nil {
    return fmt.Errorf("failed to get weather: %w", err)
}

fmt.Printf("Temperature: %.1f°C\n", weather.Temperature)
fmt.Printf("Feels like: %.1f°C\n", weather.FeelsLike)
fmt.Printf("AQI: %d (%s)\n", weather.AQI, getAQICategory(weather.AQI))
```

#### GetCurrentWeather

Gets current weather data only (no air quality).

```go
func (s *WeatherService) GetCurrentWeather(
    ctx context.Context,
    lat float64,
    lon float64,
) (*weather.WeatherData, error)
```

**Cache:** 10 minutes

#### GetForecast

Gets weather forecast (up to 5 days).

```go
func (s *WeatherService) GetForecast(
    ctx context.Context,
    lat float64,
    lon float64,
    days int,
) (*weather.ForecastData, error)
```

**Parameters:**

- `days` - Number of days (1-5)

**Cache:** 1 hour

**Example:**

```go
forecast, err := services.Weather.GetForecast(ctx, 40.7128, -74.0060, 5)
if err != nil {
    return fmt.Errorf("failed to get forecast: %w", err)
}

for _, day := range forecast.List {
    fmt.Printf("%s: %.1f°C - %s\n",
        day.DtTxt,
        day.Main.Temp,
        day.Weather[0].Description,
    )
}
```

#### GetAirQuality

Gets air quality data only.

```go
func (s *WeatherService) GetAirQuality(
    ctx context.Context,
    lat float64,
    lon float64,
) (*weather.AirQualityData, error)
```

**Cache:** 30 minutes

### Geocoding

#### GeocodeLocation

Converts location name to coordinates using OpenWeatherMap with Nominatim fallback.

```go
func (s *WeatherService) GeocodeLocation(
    ctx context.Context,
    locationName string,
) (*weather.Location, error)
```

**Returns:**

```go
type Location struct {
    Name    string
    Lat     float64
    Lon     float64
    Country string
    State   string
}
```

**Cache:** 24 hours
**Fallback:** Nominatim API if OpenWeatherMap fails

**Example:**

```go
location, err := services.Weather.GeocodeLocation(ctx, "New York, USA")
if err != nil {
    return fmt.Errorf("failed to geocode: %w", err)
}

fmt.Printf("Location: %s, %s\n", location.Name, location.Country)
fmt.Printf("Coordinates: %.4f, %.4f\n", location.Lat, location.Lon)
```

#### GetLocationName

Reverse geocodes coordinates to location name using Nominatim.

```go
func (s *WeatherService) GetLocationName(
    ctx context.Context,
    lat float64,
    lon float64,
) (string, error)
```

**Example:**

```go
name, err := services.Weather.GetLocationName(ctx, 40.7128, -74.0060)
// Returns: "New York, New York, United States"
```

### Localization

#### GetLocalizedLocationName

Returns location name in user's language with fallback to English.

```go
func (s *WeatherService) GetLocalizedLocationName(
    location *weather.Location,
    userLanguage string,
) string
```

---

## AlertService

Manages custom weather alerts and threshold monitoring.

### Constructor

```go
func NewAlertService(db *gorm.DB, redis *redis.Client) *AlertService
```

### Alert Management

#### CreateAlert

Creates a new alert configuration.

```go
func (s *AlertService) CreateAlert(
    ctx context.Context,
    userID int64,
    alertType models.AlertType,
    condition AlertCondition,
) (*models.AlertConfig, error)
```

**Alert Types:**

- `AlertTypeTemperature` - Temperature threshold
- `AlertTypeHumidity` - Humidity threshold
- `AlertTypeWindSpeed` - Wind speed threshold
- `AlertTypeAQI` - Air quality index threshold

**Condition Structure:**

```go
type AlertCondition struct {
    Operator string  // ">", "<", ">=", "<=", "=="
    Value    float64
}
```

**Example:**

```go
// Alert when temperature drops below 0°C
condition := AlertCondition{
    Operator: "<",
    Value:    0.0,
}

alert, err := services.Alert.CreateAlert(
    ctx,
    userID,
    models.AlertTypeTemperature,
    condition,
)
if err != nil {
    return fmt.Errorf("failed to create alert: %w", err)
}

fmt.Printf("Alert created: %s\n", alert.ID)
```

#### GetUserAlerts

Retrieves all active alerts for a user.

```go
func (s *AlertService) GetUserAlerts(
    ctx context.Context,
    userID int64,
) ([]models.AlertConfig, error)
```

#### CheckAlerts

Checks weather data against user's alert configurations.

```go
func (s *AlertService) CheckAlerts(
    ctx context.Context,
    weatherData *models.WeatherData,
    userID int64,
) ([]models.EnvironmentalAlert, error)
```

**Returns:** List of triggered alerts

**Example:**

```go
weatherData := &models.WeatherData{
    UserID:      userID,
    Temperature: -5.0,
    Humidity:    85,
    AQI:         150,
}

triggeredAlerts, err := services.Alert.CheckAlerts(ctx, weatherData, userID)
if err != nil {
    return fmt.Errorf("failed to check alerts: %w", err)
}

for _, alert := range triggeredAlerts {
    fmt.Printf("ALERT: %s - %s (Severity: %s)\n",
        alert.AlertType,
        alert.Message,
        alert.Severity,
    )
}
```

#### UpdateAlert

Updates an existing alert configuration.

```go
func (s *AlertService) UpdateAlert(
    ctx context.Context,
    userID int64,
    alertID uuid.UUID,
    updates map[string]interface{},
) error
```

**Updatable Fields:**

- `condition_operator`
- `condition_value`
- `is_active`
- `cooldown_minutes`

#### DeleteAlert

Marks an alert as inactive (soft delete).

```go
func (s *AlertService) DeleteAlert(
    ctx context.Context,
    userID int64,
    alertID uuid.UUID,
) error
```

#### GetAlert

Retrieves a specific alert by ID.

```go
func (s *AlertService) GetAlert(
    ctx context.Context,
    userID int64,
    alertID uuid.UUID,
) (*models.AlertConfig, error)
```

---

## SubscriptionService

Manages notification subscriptions (daily/weekly weather updates).

### Constructor

```go
func NewSubscriptionService(db *gorm.DB, redis *redis.Client) *SubscriptionService
```

### Subscription Management

#### CreateSubscription

Creates a new notification subscription.

```go
func (s *SubscriptionService) CreateSubscription(
    ctx context.Context,
    userID int64,
    subType models.SubscriptionType,
    frequency models.Frequency,
    timeOfDay string,
) (*models.Subscription, error)
```

**Subscription Types:**

- `SubscriptionTypeDaily` - Daily weather updates
- `SubscriptionTypeWeekly` - Weekly summaries
- `SubscriptionTypeAlerts` - Alert notifications
- `SubscriptionTypeExtremeWeather` - Extreme weather warnings

**Frequencies:**

- `FrequencyDaily` - Every day
- `FrequencyWeekly` - Once per week
- `FrequencyMonthly` - Once per month

**Example:**

```go
// Daily weather update at 8:00 AM user's local time
subscription, err := services.Subscription.CreateSubscription(
    ctx,
    userID,
    models.SubscriptionTypeDaily,
    models.FrequencyDaily,
    "08:00",
)
```

#### GetUserSubscriptions

Retrieves all active subscriptions for a user.

```go
func (s *SubscriptionService) GetUserSubscriptions(
    ctx context.Context,
    userID int64,
) ([]models.Subscription, error)
```

#### UpdateSubscription

Updates an existing subscription.

```go
func (s *SubscriptionService) UpdateSubscription(
    ctx context.Context,
    userID int64,
    subscriptionID uuid.UUID,
    updates map[string]interface{},
) error
```

**Updatable Fields:**

- `time_of_day`
- `frequency`
- `is_active`

#### DeleteSubscription

Marks a subscription as inactive (soft delete).

```go
func (s *SubscriptionService) DeleteSubscription(
    ctx context.Context,
    userID int64,
    subscriptionID uuid.UUID,
) error
```

#### GetActiveSubscriptions

Retrieves all active subscriptions with user data preloaded.

```go
func (s *SubscriptionService) GetActiveSubscriptions(
    ctx context.Context,
) ([]models.Subscription, error)
```

**Usage:** Called by scheduler to process notifications

#### GetSubscriptionsByType

Retrieves subscriptions filtered by type.

```go
func (s *SubscriptionService) GetSubscriptionsByType(
    ctx context.Context,
    subType models.SubscriptionType,
) ([]models.Subscription, error)
```

#### ShouldSendNotification

Determines if a notification should be sent based on time and frequency.

```go
func (s *SubscriptionService) ShouldSendNotification(
    subscription *models.Subscription,
) bool
```

**Considers:**

- Current time vs. subscription time (in user's timezone)
- Frequency (daily/weekly/monthly)
- Last sent timestamp

---

## NotificationService

Handles dual-platform notification delivery (Telegram + Slack).

### Constructor

```go
func NewNotificationService(
    config *config.IntegrationsConfig,
    logger *zerolog.Logger,
) *NotificationService
```

### Setup

#### SetBot

Sets the Telegram bot instance for sending notifications.

```go
func (s *NotificationService) SetBot(bot *gotgbot.Bot)
```

**Usage:** Called during bot initialization

```go
bot := gotgbot.NewBot(token, &gotgbot.BotOpts{})
services.Notification.SetBot(bot)
```

### Telegram Notifications

#### SendTelegramAlert

Sends alert notification to Telegram user.

```go
func (s *NotificationService) SendTelegramAlert(
    alert *models.EnvironmentalAlert,
    user *models.User,
) error
```

**Format:**

```
⚠️ Weather Alert

Type: Temperature
Condition: Temperature < 0°C
Current Value: -5.0°C
Severity: High
Location: New York, NY

Stay safe!
```

#### SendTelegramWeatherUpdate

Sends daily weather update to Telegram user.

```go
func (s *NotificationService) SendTelegramWeatherUpdate(
    weather *WeatherData,
    user *models.User,
) error
```

#### SendTelegramWeeklyUpdate

Sends weekly weather summary to Telegram user.

```go
func (s *NotificationService) SendTelegramWeeklyUpdate(
    user *models.User,
    summary string,
) error
```

### Slack Notifications

#### SendSlackAlert

Sends alert notification to Slack webhook.

```go
func (s *NotificationService) SendSlackAlert(
    alert *models.EnvironmentalAlert,
    user *models.User,
) error
```

**Format:** Slack message with attachments and color coding

#### SendSlackWeatherUpdate

Sends weather update to Slack webhook.

```go
func (s *NotificationService) SendSlackWeatherUpdate(
    weather *WeatherData,
    subscribers []models.User,
) error
```

### Error Handling

**Dual-Platform Tolerance:**

- Success if either platform succeeds
- Error only if both platforms fail
- Individual platform errors logged as warnings

**Example:**

```go
// Telegram succeeds, Slack fails → Overall success (warning logged)
// Telegram fails, Slack succeeds → Overall success (warning logged)
// Both fail → Error returned
```

---

## SchedulerService

Manages background job scheduling for alerts and notifications.

### Constructor

```go
func NewSchedulerService(
    db *gorm.DB,
    redis *redis.Client,
    weather *WeatherService,
    alert *AlertService,
    notification *NotificationService,
    logger *zerolog.Logger,
) *SchedulerService
```

### Lifecycle

#### Start

Starts the scheduler service with two concurrent jobs.

```go
func (s *SchedulerService) Start(ctx context.Context)
```

**Jobs:**

1. **Alert Processing** - Every 10 minutes
   - Fetches active users with alerts
   - Gets current weather data
   - Checks alert conditions
   - Sends notifications for triggered alerts

2. **Scheduled Notifications** - Every hour
   - Fetches active subscriptions
   - Checks if notification should be sent (timezone-aware)
   - Gets weather data
   - Sends daily/weekly updates

**Example:**

```go
ctx := context.Background()
services.StartScheduler(ctx)

// Scheduler runs in background until Stop() is called
```

#### Stop

Stops the scheduler service gracefully.

```go
func (s *SchedulerService) Stop()
```

**Usage:**

```go
defer services.Stop()
```

---

## ExportService

Provides data export functionality for GDPR compliance and user backups.

### Constructor

```go
func NewExportService(
    db *gorm.DB,
    logger *zerolog.Logger,
    localization *LocalizationService,
) *ExportService
```

### Export Formats

```go
const (
    ExportFormatJSON ExportFormat = "json"  // Machine-readable
    ExportFormatCSV  ExportFormat = "csv"   // Spreadsheet-compatible
    ExportFormatTXT  ExportFormat = "txt"   // Human-readable
)
```

### Export Types

```go
const (
    ExportTypeWeatherData    ExportType = "weather"       // Last 30 days
    ExportTypeAlerts         ExportType = "alerts"        // Configs + history (90 days)
    ExportTypeSubscriptions  ExportType = "subscriptions" // Notification preferences
    ExportTypeAll            ExportType = "all"           // Complete user data
)
```

### Export Methods

#### ExportUserData

Exports user data in specified format.

```go
func (s *ExportService) ExportUserData(
    ctx context.Context,
    userID int64,
    exportType ExportType,
    format ExportFormat,
    userLang string,
) (*bytes.Buffer, string, error)
```

**Parameters:**

- `exportType` - Type of data to export
- `format` - Output format (json, csv, txt)
- `userLang` - Language for human-readable formats

**Returns:**

- `*bytes.Buffer` - Export data buffer
- `string` - Suggested filename
- `error` - Error if export fails

**Filename Format:** `shopogoda_{type}_{username}_{date}.{ext}`

**Example:**

```go
buffer, filename, err := services.Export.ExportUserData(
    ctx,
    userID,
    services.ExportTypeAll,
    services.ExportFormatJSON,
    "en",
)
if err != nil {
    return fmt.Errorf("export failed: %w", err)
}

// Save to file
os.WriteFile(filename, buffer.Bytes(), 0644)

// Or send to user
bot.SendDocument(userID, &gotgbot.SendDocumentOpts{
    Document: gotgbot.InputFileByReader(filename, buffer),
})
```

### Export Limits

- **Weather Data:** Last 1000 records (typically 30 days)
- **Alert History:** Last 90 days
- **No limits on:** Alert configs, subscriptions, user profile

---

## LocalizationService

Provides multi-language support for the bot.

### Constructor

```go
func NewLocalizationService(logger *zerolog.Logger) *LocalizationService
```

### Initialization

#### LoadTranslations

Loads translation files from embedded filesystem.

```go
func (s *LocalizationService) LoadTranslations(localesFS fs.FS) error
```

**Usage:**

```go
//go:embed internal/locales/*.json
var localesFS embed.FS

if err := services.Localization.LoadTranslations(localesFS); err != nil {
    log.Fatal(err)
}
```

### Translation

#### T

Translates a key to the specified language with optional format arguments.

```go
func (s *LocalizationService) T(
    ctx context.Context,
    language string,
    key string,
    args ...any,
) string
```

**Example:**

```go
// Simple translation
greeting := services.Localization.T(ctx, "en-US", "welcome_message")
// Returns: "Welcome to ShoPogoda!"

// Translation with arguments
temp := services.Localization.T(ctx, "uk-UA", "temperature_reading", 25.5)
// Returns: "Температура: 25.5°C"
```

### Language Detection

#### IsLanguageSupported

Checks if a language code is supported.

```go
func (s *LocalizationService) IsLanguageSupported(language string) bool
```

**Supported Languages:** en-US, uk-UA, de-DE, fr-FR, es-ES (full IETF tags)

#### GetSupportedLanguages

Returns all supported languages with metadata.

```go
func (s *LocalizationService) GetSupportedLanguages() SupportedLanguages
```

**Returns:**

```go
type SupportedLanguages map[string]SupportedLanguage

type SupportedLanguage struct {
    Code string  // "en-US"
    Name string  // "English"
    Flag string  // "🇺🇸"
}
```

#### GetLanguageByCode

Returns language metadata by code.

```go
func (s *LocalizationService) GetLanguageByCode(
    code string,
) (SupportedLanguage, bool)
```

#### DetectLanguageFromName

Detects language code from language name or description.

```go
func (s *LocalizationService) DetectLanguageFromName(name string) string
```

**Example:**

```go
code := services.Localization.DetectLanguageFromName("English")  // "en-US"
code := services.Localization.DetectLanguageFromName("Українська")  // "uk-UA"
```

### Translation Keys

#### GetAvailableTranslationKeys

Returns all available translation keys for a language.

```go
func (s *LocalizationService) GetAvailableTranslationKeys(
    language string,
) []string
```

**Usage:** Debugging and validation

---

## DemoService

Manages demonstration data for testing and showcasing features.

### Constructor

```go
func NewDemoService(db *gorm.DB, logger *zerolog.Logger) *DemoService
```

### Demo Data Management

#### SeedDemoData

Populates database with demonstration data.

```go
func (s *DemoService) SeedDemoData(ctx context.Context) error
```

**Creates:**

- Demo user (ID: 999999999)
- Sample weather data
- Sample alerts
- Sample subscriptions

#### ClearDemoData

Removes all demonstration data.

```go
func (s *DemoService) ClearDemoData(ctx context.Context) error
```

#### ResetDemoData

Clears and re-seeds demonstration data.

```go
func (s *DemoService) ResetDemoData(ctx context.Context) error
```

### Demo User Detection

#### IsDemoUser

Checks if a user ID belongs to the demo account.

```go
func (s *DemoService) IsDemoUser(userID int64) bool
```

**Demo User ID:** 999999999

**Example:**

```go
if services.Demo.IsDemoUser(userID) {
    // Handle demo user (e.g., read-only mode, limited features)
}
```

---

## Error Handling

### Error Wrapping Pattern

All services use Go 1.13+ error wrapping with context:

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Error Checking

Use `errors.Is()` for sentinel errors and `errors.As()` for typed errors:

```go
import "errors"

if errors.Is(err, gorm.ErrRecordNotFound) {
    // Handle not found
}

var validationErr *ValidationError
if errors.As(err, &validationErr) {
    // Handle validation error
}
```

### Database Errors

Common GORM errors:

- `gorm.ErrRecordNotFound` - Entity not found
- `gorm.ErrInvalidData` - Data validation failed
- `gorm.ErrDuplicatedKey` - Unique constraint violation

### Redis Errors

Common Redis errors:

- `redis.Nil` - Key not found (cache miss)
- Connection errors - Graceful degradation (continue without cache)

### Service-Level Errors

Services return descriptive errors with context:

```go
// Good: Descriptive with context
return fmt.Errorf("failed to geocode location %q: %w", locationName, err)

// Bad: Generic error
return err
```

---

## Caching Strategy

### Cache Layers

1. **Redis** - Primary cache for frequently accessed data
2. **In-Memory** - Not currently used (all caching in Redis)

### Cache TTLs

| Data Type | TTL | Key Pattern |
|-----------|-----|-------------|
| User data | 1 hour | `user:{userID}` |
| Weather data | 10 minutes | `weather:{lat}:{lon}` |
| Forecasts | 1 hour | `forecast:{lat}:{lon}:{days}` |
| Air quality | 30 minutes | `airquality:{lat}:{lon}` |
| Geocoding | 24 hours | `geocode:{location}` |
| Reverse geocode | 24 hours | `reverse:{lat}:{lon}` |
| Activity counters | 24 hours | `stats:messages_24h`, `stats:weather_requests_24h` |

### Cache Invalidation

**Automatic:**

- TTL expiration
- User data invalidated on updates

**Manual:**

- `redis.Del(ctx, key)` for immediate invalidation

### Cache Miss Handling

All services implement graceful cache miss handling:

```go
// Try cache first
if cached, err := redis.Get(ctx, key).Result(); err == nil {
    return parseCached(cached)
}

// Cache miss - fetch from source
data := fetchFromAPI()

// Store in cache for next time
redis.Set(ctx, key, serialize(data), ttl)

return data
```

---

## Common Patterns

### Context Usage

Always pass `context.Context` as the first parameter:

```go
func (s *Service) Method(ctx context.Context, arg1, arg2 Type) error
```

**Benefits:**

- Request cancellation
- Timeout propagation
- Tracing correlation IDs

### Transaction Pattern

Use GORM transactions for multi-table operations:

```go
err := s.db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&entity1).Error; err != nil {
        return err
    }

    if err := tx.Create(&entity2).Error; err != nil {
        return err
    }

    return nil
})
```

### Preloading Relations

Use GORM's `Preload` to avoid N+1 queries:

```go
var subscriptions []models.Subscription
err := s.db.Preload("User").Find(&subscriptions).Error
```

### Timezone Handling

All times stored in UTC, converted on-demand:

```go
// Store in UTC
created_at := time.Now().UTC()

// Convert for display
localTime := services.User.ConvertToUserTime(ctx, userID, created_at)
```

### Service Injection Pattern

Services receive dependencies via constructors:

```go
type MyService struct {
    db      *gorm.DB
    redis   *redis.Client
    logger  *zerolog.Logger
    weather *WeatherService  // Inject other services
}

func NewMyService(
    db *gorm.DB,
    redis *redis.Client,
    logger *zerolog.Logger,
    weather *WeatherService,
) *MyService {
    return &MyService{
        db:      db,
        redis:   redis,
        logger:  logger,
        weather: weather,
    }
}
```

### Logging Pattern

Use structured logging with context:

```go
s.logger.Info().
    Int64("user_id", userID).
    Str("action", "weather_request").
    Float64("lat", lat).
    Float64("lon", lon).
    Msg("Processing weather request")
```

### Metrics Collection

Instrument service methods with Prometheus metrics:

```go
start := time.Now()
defer func() {
    duration := time.Since(start).Seconds()
    s.metrics.ObserveResponseTime(duration, "weather_service", "get_current")
}()
```

---

## Performance Considerations

### Response Time Targets

- **Local Development:** <200ms for cached weather queries
- **Production (Railway):** <500ms average (including cold starts)
  - Cold start: ~2-3 seconds
  - Warm requests: 200-400ms

### Database Optimization

- Indexes on frequently queried columns (user_id, created_at)
- Connection pooling (25 connections default)
- Query optimization for large datasets
- Embedded user locations (no joins required)

### Cache Optimization

- Aggressive caching for expensive operations (geocoding, weather API)
- Cache warming for frequently accessed data
- Monitor cache hit rates via Prometheus

### API Rate Limiting

- OpenWeatherMap: 1000 calls/day (free tier)
- Use caching to stay within limits
- Implement exponential backoff on failures

---

## Testing

### Unit Testing

Services use concrete types (no interfaces) with mock infrastructure:

```go
import "github.com/valpere/shopogoda/tests/helpers"

func TestUserService_GetUser(t *testing.T) {
    mockDB := helpers.NewMockDB(t)
    defer mockDB.Close()

    mockRedis := helpers.NewMockRedis()
    metricsCollector := metrics.New()

    service := NewUserService(mockDB.DB, mockRedis.Client, metricsCollector, time.Now())

    // Test implementation
}
```

### Integration Testing

Integration tests use testcontainers for real database/Redis:

```go
func TestIntegration(t *testing.T) {
    ctx := context.Background()

    // Start PostgreSQL container
    postgres := testcontainers.PostgreSQL(ctx, t)
    db := postgres.Connect()

    // Start Redis container
    redis := testcontainers.Redis(ctx, t)
    client := redis.Connect()

    // Run tests with real infrastructure
}
```

### Test Coverage

- **Current:** 33.7% overall, 40.5% for testable packages
- **Target:** 80% for service layer
- See [Testing Guide](TESTING.md) for details

---

## Migration Guide

### From Older Versions

#### Location Model Change (v0.1.0 → v0.1.1)

**Before:**

```go
// Multiple locations per user with separate Location entity
location, err := services.Location.GetDefaultLocation(ctx, userID)
```

**After:**

```go
// Single embedded location per user
name, lat, lon, err := services.User.GetUserLocation(ctx, userID)
```

**Migration Steps:**

1. Run database migration to embed location fields in User table
2. Update code to use `UserService` methods instead of `LocationService`
3. Remove `LocationService` references

#### Timezone Independence (v0.1.1)

**Important Changes:**

- `SetUserLocation()` no longer resets timezone to UTC
- `GetUserTimezone()` returns user's timezone regardless of location status
- Location and timezone are completely independent settings

---

## Examples

### Complete Weather Flow

```go
package main

import (
    "context"
    "fmt"
    "github.com/valpere/shopogoda/internal/services"
)

func GetWeatherForUser(ctx context.Context, svcs *services.Services, userID int64) error {
    // 1. Get user's location
    locationName, lat, lon, err := svcs.User.GetUserLocation(ctx, userID)
    if err != nil {
        return fmt.Errorf("user has no location set: %w", err)
    }

    // 2. Get weather data (with caching)
    weather, err := svcs.Weather.GetCompleteWeatherData(ctx, lat, lon)
    if err != nil {
        return fmt.Errorf("failed to get weather: %w", err)
    }

    // 3. Check for alerts
    user, _ := svcs.User.GetUser(ctx, userID)
    weatherModel := weather.ToModelWeatherData()
    weatherModel.UserID = userID

    alerts, err := svcs.Alert.CheckAlerts(ctx, weatherModel, userID)
    if err != nil {
        return fmt.Errorf("failed to check alerts: %w", err)
    }

    // 4. Send notifications for triggered alerts
    for _, alert := range alerts {
        if err := svcs.Notification.SendTelegramAlert(&alert, user); err != nil {
            fmt.Printf("Failed to send alert: %v\n", err)
        }
    }

    // 5. Convert time to user's timezone
    localSunrise := svcs.User.ConvertToUserTime(ctx, userID, weather.Sunrise)

    fmt.Printf("Weather in %s:\n", locationName)
    fmt.Printf("Temperature: %.1f°C (feels like %.1f°C)\n", weather.Temperature, weather.FeelsLike)
    fmt.Printf("Conditions: %s\n", weather.Description)
    fmt.Printf("AQI: %d\n", weather.AQI)
    fmt.Printf("Sunrise: %s\n", localSunrise.Format("15:04"))

    return nil
}
```

### Creating a Complete Notification Subscription

```go
func SetupDailyWeather(ctx context.Context, svcs *services.Services, userID int64) error {
    // 1. Create daily subscription
    sub, err := svcs.Subscription.CreateSubscription(
        ctx,
        userID,
        models.SubscriptionTypeDaily,
        models.FrequencyDaily,
        "08:00",  // 8 AM in user's local time
    )
    if err != nil {
        return fmt.Errorf("failed to create subscription: %w", err)
    }

    fmt.Printf("Subscription created: %s\n", sub.ID)
    fmt.Printf("You'll receive daily weather at 08:00 %s\n",
        svcs.User.GetUserTimezone(ctx, userID))

    return nil
}
```

### Exporting User Data

```go
func ExportAllData(ctx context.Context, svcs *services.Services, userID int64) error {
    // Get user language
    user, err := svcs.User.GetUser(ctx, userID)
    if err != nil {
        return err
    }

    // Export all data as JSON
    buffer, filename, err := svcs.Export.ExportUserData(
        ctx,
        userID,
        services.ExportTypeAll,
        services.ExportFormatJSON,
        user.Language,
    )
    if err != nil {
        return fmt.Errorf("export failed: %w", err)
    }

    // Save to file
    if err := os.WriteFile(filename, buffer.Bytes(), 0644); err != nil {
        return fmt.Errorf("failed to save file: %w", err)
    }

    fmt.Printf("Data exported to: %s\n", filename)
    return nil
}
```

---

## Support

For questions or issues with the API:

- **Documentation:** [docs/](.)
- **Issues:** [GitHub Issues](https://github.com/valpere/shopogoda/issues)

---

**Last Updated:** 2025-10-11
**API Version:** 0.1.2-dev
**Go Version:** 1.23+
