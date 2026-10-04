# ShoPogoda Demo Guide

Complete guide for getting ShoPogoda running locally and using demo mode for testing, presentations, and development.

## Table of Contents

- [Quick Start (5 Minutes)](#quick-start-5-minutes)
- [Demo Mode](#demo-mode)
- [Available Commands](#available-commands)
- [Monitoring Stack](#monitoring-stack)
- [Development Workflow](#development-workflow)
- [Troubleshooting](#troubleshooting)

---

## Quick Start (5 Minutes)

Get the ShoPogoda weather bot running locally in under 5 minutes!

### Prerequisites

- **Docker & Docker Compose** (optional, only for the Prometheus/Grafana/Jaeger monitoring stack)
- **Go 1.24.6** (for building the bot)
- **Telegram Account** (to interact with your bot)
- **API Keys**:
  - Telegram Bot Token from [@BotFather](https://t.me/BotFather)
  - OpenWeatherMap API Key from [openweathermap.org](https://openweathermap.org/api)

### Setup Steps

#### 1. Clone the Repository

```bash
git clone https://github.com/valpere/shopogoda.git
cd shopogoda
```

#### 2. Initialize the Project

```bash
make init
```

This command:

- Creates `.env` from `.env.example`
- Starts the optional monitoring containers (Prometheus, Grafana, Jaeger)
- The SQLite database file (`./data/shopogoda.db`) is created and migrated automatically on first run

#### 3. Configure Your Bot

Edit `.env` and add your API keys:

```bash
# Required: Get from @BotFather on Telegram
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz

# Required: Get from openweathermap.org
OPENWEATHER_API_KEY=abcdef1234567890abcdef1234567890

# Optional: For enterprise notifications
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Optional: Enable demo mode for testing
DEMO_MODE=true
```

#### 4. Start the Bot

```bash
make run
```

Expected output:

```plaintext
✓ Database connected
✓ Bot initialized
✓ Listening for updates...
```

#### 5. Try It Out!

Open Telegram and find your bot, then try:

```plaintext
/start          - Welcome message and setup
/weather        - Get current weather
/forecast       - 5-day weather forecast
/air            - Air quality information
/setlocation    - Set your location
/settings       - Configure preferences
```

---

## Demo Mode

ShoPogoda includes a comprehensive demo mode for testing, presentations, and evaluation purposes. Demo mode automatically populates the database with realistic demonstration data.

### Enable Demo Mode

Add to your `.env` file:

```bash
DEMO_MODE=true
```

When the bot starts with demo mode enabled, it automatically creates:

- ✅ **Demo User** (ID: 999999999)
- ✅ **24 hours of weather data** with realistic patterns
- ✅ **3 alert configurations** (temperature, humidity, air quality)
- ✅ **3 notification subscriptions** (daily, weekly, alerts)

### Demo User Details

| Property | Value |
|----------|-------|
| **Telegram ID** | 999999999 |
| **Username** | demo_user |
| **Name** | Demo User |
| **Location** | Kyiv, Ukraine (50.4501°N, 30.5234°E) |
| **Timezone** | Europe/Kyiv |
| **Language** | English |
| **Units** | Metric |
| **Role** | User |

### Seeded Data

#### Weather Data (24 hours)

- **Records**: Hourly weather data for the last 24 hours
- **Temperature**: Realistic variation (10-25°C with sine wave pattern)
- **Conditions**: Automatically varied (Freezing, Cold, Cool, Mild, Warm, Hot)
- **Wind**: 5-13 km/h with varying direction
- **Humidity**: 60-80%
- **Pressure**: 1013-1018 hPa
- **Air Quality**: Last 6 hours include AQI data (50-80 range)

#### Alert Configurations (3 alerts)

1. **Temperature Alert**
   - Type: Temperature
   - Condition: Greater than
   - Threshold: 30.0°C
   - Status: Active ✅

2. **Humidity Alert**
   - Type: Humidity
   - Condition: Greater than
   - Threshold: 80.0%
   - Status: Active ✅

3. **Air Quality Alert**
   - Type: Air Quality (AQI)
   - Condition: Greater than
   - Threshold: 100
   - Status: Inactive ⏸️

#### Notification Subscriptions (3 subscriptions)

1. **Daily Weather Update**
   - Type: Daily
   - Frequency: Daily
   - Delivery Time: 08:00
   - Status: Active ✅

2. **Weekly Forecast**
   - Type: Weekly
   - Frequency: Weekly
   - Delivery Time: 09:00
   - Status: Active ✅

3. **Alert Notifications**
   - Type: Alerts
   - Frequency: Hourly
   - Status: Active ✅

### Admin Commands

Demo mode includes admin commands for managing demonstration data:

#### Reset Demo Data

```
/demoreset
```

**Admin only** - Clears existing demo data and re-seeds with fresh data.

Use this when:
- Demo data becomes outdated
- Testing data migrations
- Preparing for presentations
- Resetting after testing

#### Clear Demo Data

```
/democlear
```

**Admin only** - Removes all demo data from the database.

Use this when:
- Disabling demo mode
- Cleaning up test environment
- Preparing for production deployment

### Use Cases

#### 1. Development & Testing

Enable demo mode during development to have consistent test data:

```bash
DEMO_MODE=true make dev
```

Benefits:
- Immediate data availability
- Consistent test scenarios
- No manual data entry
- Faster iteration

#### 2. Demonstrations & Presentations

Showcase bot features with realistic data:

1. Enable demo mode
2. Start bot
3. Show /weather, /forecast, /air commands
4. Display configured alerts and subscriptions
5. Demonstrate export functionality

#### 3. Integration Testing

Use demo data for automated testing:

```go
// In tests
if cfg.Bot.DemoMode {
    // Test against known demo user
    userID := services.DemoUserID
    // Run test scenarios
}
```

#### 4. Documentation Screenshots

Generate consistent screenshots for documentation:

1. Enable demo mode
2. Interact with demo user
3. Capture screenshots
4. Reset with `/demoreset` for fresh state

### Demo Service Architecture

Location: `internal/services/demo_service.go`

```go
type DemoService struct {
    db     *gorm.DB
    logger *zerolog.Logger
}

// Key methods
func (s *DemoService) SeedDemoData(ctx context.Context) error
func (s *DemoService) ClearDemoData(ctx context.Context) error
func (s *DemoService) ResetDemoData(ctx context.Context) error
func (s *DemoService) IsDemoUser(userID int64) bool
```

### Best Practices

#### Development

✅ **DO**:
- Use demo mode for consistent test data
- Reset demo data before presentations
- Document any custom demo scenarios

❌ **DON'T**:
- Enable demo mode in production
- Modify demo user ID (999999999)
- Rely on demo data for real user testing

#### Production

⚠️ **IMPORTANT**: Always disable demo mode in production:

```bash
DEMO_MODE=false
```

Demo mode is for development and demonstration only.

---

## Available Commands

### Demo Features to Try

#### Basic Weather Queries

1. **Set Your Location**:

   ```plaintext
   /setlocation
   → Select "📍 Share Location" or enter "Kyiv"
   ```

2. **Get Current Weather**:

   ```plaintext
   /weather
   → See temperature, humidity, wind, conditions
   ```

3. **View Forecast**:

   ```plaintext
   /forecast
   → 5-day weather prediction
   ```

#### Advanced Features

4. **Air Quality Monitoring**:

   ```plaintext
   /air
   → AQI index and pollutant levels
   ```

5. **Custom Alerts**:

   ```plaintext
   /addalert
   → Set temperature threshold alerts
   → Example: Alert when temp > 30°C
   ```

6. **Scheduled Notifications**:

   ```plaintext
   /subscribe
   → Daily weather at 8:00 AM
   → Weekly forecast on Sunday
   ```

7. **Multi-Language Support**:

   ```plaintext
   /settings → 🌐 Language
   → Choose: English, Ukrainian, German, French, Spanish
   ```

8. **Data Export**:

   ```plaintext
   /settings → 📊 Data Export
   → Export your weather data, alerts, subscriptions
   → Formats: JSON, CSV, TXT
   ```

---

## Monitoring Stack

Access the monitoring dashboards:

| Service | URL | Credentials |
|---------|-----|-------------|
| **Bot Health** | http://localhost:8080/health | - |
| **Metrics** | http://localhost:8080/metrics | - |
| **Grafana** | http://localhost:3000 | admin / admin123 |
| **Prometheus** | http://localhost:9090 | - |
| **Jaeger Tracing** | http://localhost:16686 | - |

---

## Architecture Overview

```plaintext
┌─────────────┐
│   Telegram  │
│     Bot     │
└──────┬──────┘
       │
       v
┌─────────────────────────────────────┐
│         ShoPogoda Bot               │
│  ┌────────────┐  ┌──────────────┐  │
│  │  Handlers  │  │   Services   │  │
│  │ (Commands) │→ │ (Business    │  │
│  └────────────┘  │   Logic)     │  │
│                  └──────┬───────┘  │
│                         v           │
│  ┌──────────┐    ┌──────────────┐  │
│  │ In-proc  │←── │    SQLite    │  │
│  │ (Cache)  │    │  (Storage)   │  │
│  └──────────┘    └──────────────┘  │
└─────────────────────────────────────┘
       │
       v
┌─────────────────────────────────────┐
│     External APIs                   │
│  • OpenWeatherMap (weather data)    │
│  • Slack/Teams (notifications)      │
└─────────────────────────────────────┘
```

---

## Development Workflow

### Available Commands

```bash
# Development
make dev              # Start all services + build bot
make run              # Build and run the bot
make build            # Build the application

# Testing
make test             # Run unit tests
make test-coverage    # Generate coverage report
make test-integration # Run integration tests

# Code Quality
make lint             # Run golangci-lint
make fmt              # Format code with gofmt

# Database
make migrate          # Run migrations
make migrate-rollback # Rollback last migration

# Docker
make docker-up        # Start optional monitoring stack (Prometheus, Grafana, Jaeger)
make docker-down      # Stop monitoring containers
make docker-logs      # View monitoring container logs
make docker-build     # Build production image

# Cleanup
make clean            # Remove build artifacts
```

### Project Structure

```plaintext
shopogoda/
├── cmd/bot/              # Application entry point
├── internal/             # Private application code
│   ├── bot/             # Bot initialization
│   ├── config/          # Configuration management
│   ├── database/        # SQLite connection and in-process cache
│   ├── handlers/
│   │   ├── commands/    # Telegram command handlers
│   │   └── callbacks/   # Callback query handlers
│   ├── locales/         # Translation files (5 languages)
│   ├── middleware/      # Logging, metrics, auth, rate limiting
│   ├── models/          # GORM models
│   └── services/        # Business logic services
├── pkg/
│   ├── metrics/         # Prometheus metrics
│   └── weather/         # Weather API clients
├── docs/                # Documentation
├── deployments/         # Docker, K8s configs
└── tests/               # Integration and E2E tests
```

---

## Configuration Options

ShoPogoda uses a hierarchical configuration system:

1. **Environment Variables** (highest priority)
2. **`.env` file** in current directory
3. **YAML configuration files**:
   - `./shopogoda.yaml`
   - `~/.shopogoda.yaml`
   - `~/.config/shopogoda.yaml`
   - `/etc/shopogoda.yaml`
4. **Built-in defaults** (lowest priority)

See [CONFIGURATION.md](CONFIGURATION.md) for complete reference.

---

## Troubleshooting

### Bot doesn't respond

**Check bot is running:**

```bash
curl http://localhost:8080/health
# Should return: {"status":"healthy"}
```

**Check logs:**

```bash
# In terminal where bot is running, look for errors
# Common issues:
# - Invalid TELEGRAM_BOT_TOKEN
# - Invalid OPENWEATHER_API_KEY
# - Database file not writable (check DB_PATH)
```

### Database errors

**Check the database path is writable:**

```bash
# Default location (override with DB_PATH in .env)
ls -la ./data/
```

The bot creates `./data/shopogoda.db` on first start; the `data` directory must be writable. To start from a clean database: `make db-reset` (destroys all data).

### Weather API errors

**Verify OpenWeatherMap API key:**

```bash
# Test API key manually:
curl "https://api.openweathermap.org/data/2.5/weather?q=London&appid=YOUR_API_KEY"
```

**Check rate limits:**

- Free tier: 60 calls/minute, 1,000,000 calls/month
- Upgrade if needed at openweathermap.org

### Stale data after changes

The cache is in-process (no external service) and is emptied on every restart. If a manual database edit does not show up, restart the bot.

### Demo data not appearing

**Check configuration:**
```bash
# Verify DEMO_MODE is set
grep DEMO_MODE .env

# Check logs for demo mode initialization
./shopogoda | grep -i demo
```

**Reset demo data manually:**
```
/demoreset  # As admin user
```

### Demo user already exists

Demo mode uses `FirstOrCreate` - it won't duplicate the demo user. If demo data seems incomplete:

1. Clear existing demo data: `/democlear`
2. Re-seed fresh data: `/demoreset`

### Permission denied for admin commands

Admin commands (`/demoreset`, `/democlear`) require admin role. Check user role:

```sql
SELECT telegram_id, username, role FROM users WHERE telegram_id = YOUR_ID;
```

Update role if needed:

```sql
UPDATE users SET role = 'admin' WHERE telegram_id = YOUR_ID;
```

---

## Next Steps

### For Users

- Explore all bot commands: `/help`
- Set up custom alerts: `/addalert`
- Configure notifications: `/subscribe`
- Export your data: `/settings` → Data Export

### For Developers

- Read [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment
- Check [CODE_QUALITY.md](CODE_QUALITY.md) for contribution guidelines
- Review [ROADMAP.md](ROADMAP.md) for upcoming features
- See [ARCHITECTURE.md](ARCHITECTURE.md) for system design

### For Enterprise

- Set up Slack integration (add `SLACK_WEBHOOK_URL` to `.env`)
- Configure role-based access control
- Deploy to cloud (see [DEPLOYMENT.md](DEPLOYMENT.md))
- Set up custom Grafana dashboards

---

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/valpere/shopogoda/issues)
- **Discussions**: [GitHub Discussions](https://github.com/valpere/shopogoda/discussions)

---

## License

MIT License - see [LICENSE](../LICENSE) for details.

---

**Ready to deploy?** Check out [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment guides!
