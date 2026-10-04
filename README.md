# ShoPogoda (Що Погода) - Enterprise Weather Bot

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/valpere/shopogoda)](https://goreportcard.com/report/github.com/valpere/shopogoda)
[![codecov](https://codecov.io/gh/valpere/shopogoda/branch/main/graph/badge.svg)](https://codecov.io/gh/valpere/shopogoda)
[![GitHub release](https://img.shields.io/github/release/valpere/shopogoda.svg)](https://github.com/valpere/shopogoda/releases)

A production-ready Telegram bot for weather monitoring, environmental alerts, and enterprise integrations. Runs as a single binary or one container, with state in a single SQLite file and an in-process cache; no external database or cache services required.

## Live Demo

- **🇺🇦 Ukrainian (uk)**: [Жива демонстрація](https://t.me/shopogoda_bot)
- **🇺🇸 English (en)**: [Live demonstration](https://t.me/shopogoda_bot)
- **🇩🇪 German (de)**: [Live-Demonstration](https://t.me/shopogoda_bot)
- **🇫🇷 French (fr)**: [Démonstration en direct](https://t.me/shopogoda_bot)
- **🇪🇸 Spanish (es)**: [Demostración en vivo](https://t.me/shopogoda_bot)

## 🌟 Features

### Core Weather Services
- **Real-time Weather Data**: Current conditions with comprehensive metrics
- **5-Day Forecasts**: Detailed weather predictions
- **Air Quality Monitoring**: AQI and pollutant tracking
- **Smart Location Management**: Single location per user with GPS and name-based input
- **Multi-language Support**: Complete localization in Ukrainian, English, German, French, and Spanish with dynamic language switching

### Enterprise Features
- **Advanced Alert System**: Custom thresholds with interactive management (edit, toggle, delete)
- **Slack/Teams Integration**: Automated notifications
- **Role-Based Access Control**: Admin, moderator, and user roles
- **Monitoring & Analytics**: Prometheus metrics and Grafana dashboards
- **Simple Operations**: Single instance, SQLite storage, in-process cache (no external services)

### Technical Excellence
- **Scalable Architecture**: Microservices-ready design
- **Comprehensive Testing**: 34.2% coverage (39.4% on testable packages) with unit, integration, and bot mock tests
- **Production Ready**: Docker containerization and CI/CD
- **Enterprise Security**: Rate limiting, input validation, audit logs

## 🚀 Quick Start

### Prerequisites

**For Local Development:**
- Go 1.25+
- Docker & Docker Compose (optional, only for the Prometheus/Grafana/Jaeger stack)
- Telegram Bot Token (from @BotFather)
- OpenWeatherMap API Key

**For Production Deployment:**
- A host that can run one instance (binary or container) with a persistent volume for the SQLite file
- A public HTTPS URL if using webhook mode (polling also works)

### Setup Commands

```bash
# 1. Clone and initialize
git clone https://github.com/valpere/shopogoda.git
cd shopogoda
make init

# 2. Configure environment
cp .env.example .env
# Edit .env with your API keys

# 3. Start development environment
make dev

# 4. Run the bot
make run
```

**📖 For detailed demo setup instructions:** [DEMO_GUIDE.md](docs/DEMO_GUIDE.md)

### Configuration

ShoPogoda supports multiple configuration methods for different deployment scenarios:

#### Quick Setup (Development)

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Edit `.env` with your credentials:
```bash
# Required
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
OPENWEATHER_API_KEY=your_openweather_api_key

# Optional
DB_PATH=./data/shopogoda.db
SLACK_WEBHOOK_URL=your_slack_webhook_url
BOT_DEBUG=true
LOG_LEVEL=debug
```

#### Configuration Methods

ShoPogoda uses a hierarchical configuration system with the following precedence:

1. **Environment Variables** (highest priority)
2. **.env file** in current directory
3. **YAML configuration files** (first found):
   - `./shopogoda.yaml` (current directory)
   - `~/.shopogoda.yaml` (home directory)
   - `~/.config/shopogoda.yaml` (user config directory)
   - `/etc/shopogoda.yaml` (system-wide)
4. **Built-in defaults** (lowest priority)

#### YAML Configuration

For production deployments, create a `shopogoda.yaml` file:

```yaml
# Bot configuration
bot:
  debug: false
  webhook_url: "https://yourdomain.com/webhook"
  webhook_port: 8080

# Database configuration
database:
  host: localhost
  port: 5432
  user: shopogoda
  name: shopogoda
  ssl_mode: require  # Enable for production

# Weather service configuration
weather:
  user_agent: "ShoPogoda-Weather-Bot/1.0 (your-contact@example.com)"

# Logging configuration
logging:
  level: info
  format: json
```

**📖 For complete configuration reference, deployment examples, and troubleshooting:** [Configuration Guide](docs/CONFIGURATION.md)

## 📊 Monitoring

Access the monitoring stack:

- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Jaeger Tracing**: http://localhost:16686
- **Bot Health**: http://localhost:8080/health
- **Metrics**: http://localhost:8080/metrics

## 🏗️ Architecture

```
shopogoda/
├── cmd/bot/              # Application entrypoints
├── internal/             # Private application code
│   ├── bot/             # Bot initialization and setup
│   ├── config/          # Configuration management
│   ├── database/        # Database connections
│   ├── handlers/        # Telegram command handlers
│   ├── locales/         # Translation files (en, de, es, fr, uk)
│   ├── middleware/      # Bot middleware (auth, logging)
│   ├── models/          # Data models and structs
│   └── services/        # Business logic services (incl. localization)
├── pkg/                 # Public libraries
│   ├── weather/         # Weather API clients
│   ├── alerts/          # Alert engine
│   └── metrics/         # Prometheus metrics
└── deployments/         # Docker, K8s configurations
```

## 🔧 Development

### Available Commands

```bash
make help           # Show all available commands
make deps           # Install dependencies
make build          # Build the application
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linter
make docker-build   # Build Docker image
make migrate        # Run database migrations
```

## 🚀 Deployment

### Production Deployment

ShoPogoda runs as **exactly one instance** (SQLite single writer): a single binary, or one container with a volume mounted at `/app/data`.

- **Docker Compose**: `docker/docker-compose.prod.yml` / `docker-compose.staging.yml` run the bot with optional Prometheus/Grafana/Jaeger and a named data volume
- **Kubernetes**: `deployments/k8s/` (`replicas: 1`, strategy `Recreate`, PVC)
- **Backups**: `sqlite3 <file> ".backup out.db"` or Litestream

**📖 Complete deployment guide:** [DEPLOYMENT.md](docs/DEPLOYMENT.md)

## 📈 Performance

- **Caching**: In-process TTL cache (weather 10m, forecast 1h, air quality 30m, geocoding 24h); lost on restart
- **Database**: SQLite in WAL mode; optimized composite indexes
- **Latency / hit rate**: depend on your host; measure on your deployment

## 🔒 Security

- **Input Validation**: Comprehensive validation and sanitization of all user inputs
- **Rate Limiting**: 10 requests/minute per user to prevent abuse (in-process)
- **SQL Injection Prevention**: GORM ORM with parameterized queries
- **Secure Credential Management**: Environment variables and secret management
- **Audit Logging**: Complete audit trail for compliance and monitoring

## 🌐 Multi-Language Support

ShoPogoda offers comprehensive internationalization with complete localization in 5 languages:

- **🇺🇦 Ukrainian (uk)**: Native support with cultural considerations
- **🇺🇸 English (en)**: Default language with comprehensive coverage
- **🇩🇪 German (de)**: Full localization for German-speaking regions
- **🇫🇷 French (fr)**: Complete French translation
- **🇪🇸 Spanish (es)**: Full Spanish localization

### Language Features
- **Dynamic Language Switching**: Users can change language anytime via `/settings`
- **Persistent Preferences**: Language settings are saved and preserved across sessions
- **Complete UI Localization**: All bot messages, buttons, and help text translated
- **Timezone Independence**: Language and timezone settings are managed separately
- **Fallback System**: Automatic fallback to English for missing translations

## 📚 Documentation

### User Guides
- **[Demo Guide](docs/DEMO_GUIDE.md)** - Get started in 5 minutes with demo mode
- **[Configuration Guide](docs/CONFIGURATION.md)** - Complete configuration reference
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment instructions
- **[Admin Setup Guide](docs/ADMIN_SETUP.md)** - Grant admin access to bot owners

### Developer Documentation
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to the project
- **[Architecture](docs/ARCHITECTURE.md)** - System architecture and design
- **[API Reference](docs/API_REFERENCE.md)** - Complete service layer API documentation
- **[Testing Guide](docs/TESTING.md)** - Comprehensive testing documentation (34.2% overall, 39.4% testable packages)
- **[Code Quality Guidelines](docs/CODE_QUALITY.md)** - Contribution standards

### Project Management
- **[Roadmap](docs/ROADMAP.md)** - Feature roadmap and future plans
- **[Release Process](docs/RELEASE_PROCESS.md)** - Release management guide
- **[Changelog](CHANGELOG.md)** - Version history and changes

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

For enterprise support and custom development:
- LinkedIn: [valentynsolomko](https://linkedin.com/in/valentynsolomko)
- GitHub: [valpere](https://github.com/valpere)

---

**Built with ❤️ by Valentyn Solomko - Senior Backend Engineering Leader**
