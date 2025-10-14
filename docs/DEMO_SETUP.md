# ShoPogoda Demo Setup Guide

Get the ShoPogoda weather bot running locally in under 5 minutes!

## Prerequisites

- **Docker & Docker Compose** (for PostgreSQL, Redis, monitoring)
- **Go 1.24.6** (for building the bot)
- **Telegram Account** (to interact with your bot)
- **API Keys**:
  - Telegram Bot Token from [@BotFather](https://t.me/BotFather)
  - OpenWeatherMap API Key from [openweathermap.org](https://openweathermap.org/api)

## Quick Start (5 Minutes)

### 1. Clone the Repository

```bash
git clone https://github.com/valpere/shopogoda.git
cd shopogoda
```

### 2. Initialize the Project

```bash
make init
```

This command:

- Creates `.env` from `.env.example`
- Starts PostgreSQL, Redis, Prometheus, Grafana containers
- Applies database migrations

### 3. Configure Your Bot

Edit `.env` and add your API keys:

```bash
# Required: Get from @BotFather on Telegram
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz

# Required: Get from openweathermap.org
OPENWEATHER_API_KEY=abcdef1234567890abcdef1234567890

# Optional: For enterprise notifications
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

### 4. Start the Bot

```bash
make run
```

Expected output:

```plaintext
✓ Database connected
✓ Redis connected
✓ Bot initialized
✓ Listening for updates...
```

### 5. Try It Out!

Open Telegram and find your bot, then try:

```plaintext
/start          - Welcome message and setup
/weather        - Get current weather
/forecast       - 5-day weather forecast
/air            - Air quality information
/setlocation    - Set your location
/settings       - Configure preferences
```

## Demo Features to Try

### Basic Weather Queries

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

### Advanced Features

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

## Monitoring Stack

Access the monitoring dashboards:

| Service | URL | Credentials |
|---------|-----|-------------|
| **Bot Health** | http://localhost:8080/health | - |
| **Metrics** | http://localhost:8080/metrics | - |
| **Grafana** | http://localhost:3000 | admin / admin123 |
| **Prometheus** | http://localhost:9090 | - |
| **Jaeger Tracing** | http://localhost:16686 | - |

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
│  │  Redis   │←── │  PostgreSQL  │  │
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
make docker-up        # Start all containers
make docker-down      # Stop all containers
make docker-logs      # View container logs
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
│   ├── database/        # DB connections (PostgreSQL, Redis)
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
# - Database connection failure
```

### Database connection failed

**Ensure containers are running:**

```bash
docker ps
# Should show: postgres, redis, prometheus, grafana

# If not running:
make docker-up
```

**Check database credentials in `.env`:**

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=shopogoda
DB_PASSWORD=your_password
DB_NAME=shopogoda
```

### Weather API errors

**Verify OpenWeatherMap API key:**

```bash
# Test API key manually:
curl "https://api.openweathermap.org/data/2.5/weather?q=London&appid=YOUR_API_KEY"
```

**Check rate limits:**

- Free tier: 60 calls/minute, 1,000,000 calls/month
- Upgrade if needed at openweathermap.org

### Redis connection issues

**Check Redis container:**

```bash
docker ps | grep redis
docker logs shopogoda-redis

# Test Redis connection:
docker exec -it shopogoda-redis redis-cli ping
# Should return: PONG
```

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

### For Enterprise

- Set up Slack integration (add `SLACK_WEBHOOK_URL` to `.env`)
- Configure role-based access control
- Deploy to cloud (see [DEPLOYMENT.md](DEPLOYMENT.md))
- Set up custom Grafana dashboards

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/valpere/shopogoda/issues)
- **Discussions**: [GitHub Discussions](https://github.com/valpere/shopogoda/discussions)

## License

MIT License - see [LICENSE](../LICENSE) for details.

---

**Ready to deploy?** Check out [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment guides!
