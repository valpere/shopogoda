# ShoPogoda (Що Погода) - Enterprise Weather Bot

A production-ready Telegram bot for weather monitoring, environmental alerts, and enterprise integrations. Currently deployed on Railway with Supabase PostgreSQL and Upstash Redis.

## 🌟 Features

### Core Weather Services
- **Real-time Weather Data**: Current conditions with comprehensive metrics
- **5-Day Forecasts**: Detailed weather predictions
- **Air Quality Monitoring**: AQI and pollutant tracking
- **Smart Location Management**: Single location per user with GPS and name-based input
- **Multi-language Support**: Complete localization in Ukrainian, English, German, French, and Spanish with dynamic language switching

### Enterprise Features
- **Advanced Alert System**: Custom thresholds and conditions
- **Slack/Teams Integration**: Automated notifications
- **Role-Based Access Control**: Admin, moderator, and user roles
- **Monitoring & Analytics**: Prometheus metrics and Grafana dashboards
- **High Availability**: Redis caching and PostgreSQL clustering

### Technical Excellence
- **Scalable Architecture**: Microservices-ready design
- **Comprehensive Testing**: 30.5% coverage with unit, integration, and bot mock tests
- **Production Ready**: Docker containerization and CI/CD
- **Enterprise Security**: Rate limiting, input validation, audit logs

## 🚀 Quick Start

### Prerequisites

**For Local Development:**
- Go 1.24+
- Docker & Docker Compose
- Telegram Bot Token (from @BotFather)
- OpenWeatherMap API Key

**For Production Deployment:**
- Railway account (free tier available)
- Supabase account (PostgreSQL - free tier)
- Upstash account (Redis - free tier)

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

**📖 For detailed demo setup instructions:** [DEMO_SETUP.md](docs/DEMO_SETUP.md)

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

### Production Deployment (Railway)

**Currently deployed and running in production:**

```bash
# Quick deploy to Railway (recommended)
railway login
railway init
railway up

# Configure environment variables in Railway dashboard
# See docs/DEPLOYMENT_RAILWAY.md for complete guide
```

**Live Production:**
- Health: https://shopogoda-svc-production.up.railway.app/health
- Stack: Railway + Supabase (PostgreSQL) + Upstash (Redis)
- Cost: $0/month (free tier)
- Status: ✅ Production-ready

**📖 Complete deployment guide:** [DEPLOYMENT_RAILWAY.md](docs/DEPLOYMENT_RAILWAY.md)

### Alternative Deployment Options

The bot supports multiple platforms:
- **Railway** - Primary production platform (free tier, 500 hrs/month)
- **Vercel** - Serverless functions (free tier, 100GB bandwidth/month)
- **Fly.io** - Global edge deployment (~$5-10/month)
- **Replit** - All-in-one IDE (free with sleep, $20/month always-on)
- **Docker** - Traditional container deployment (any cloud)

**📖 Platform comparison:** [DEPLOYMENT.md](docs/DEPLOYMENT.md)

## 📈 Performance

**Production Metrics (Railway deployment):**
- **Response Time**: < 500ms average (including cold starts)
- **Uptime**: 99.5%+ on free tier
- **Cache Hit Rate**: > 85% (Upstash Redis)
- **Database Latency**: 100-200ms (Supabase pooler)
- **Database Indexes**: Optimized composite indexes for 2-3x faster queries

**Free Tier Limits:**
- Railway: 500 execution hours/month (continuous uptime: ~20.8 days; webhook mode requires always-on)
- Supabase: 500MB storage, 2GB bandwidth/month
- Upstash: 10,000 commands/day (~6.9 commands/minute on average; actual usage varies by traffic patterns)

## 🔒 Security

- **Row Level Security (RLS)**: Supabase PostgREST API secured with deny-by-default policies
- **Input Validation**: Comprehensive validation and sanitization of all user inputs
- **Rate Limiting**: 10 requests/minute per user to prevent abuse
- **SQL Injection Prevention**: GORM ORM with parameterized queries
- **Secure Credential Management**: Environment variables and secret management
- **Audit Logging**: Complete audit trail for compliance and monitoring

**📖 Security documentation:** [DATABASE_SECURITY.md](docs/DATABASE_SECURITY.md)

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
- **[Demo Setup Guide](docs/DEMO_SETUP.md)** - Get started in 5 minutes
- **[Configuration Guide](docs/CONFIGURATION.md)** - Complete configuration reference
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment instructions

### Developer Documentation
- **[API Reference](docs/API_REFERENCE.md)** - Complete service layer API documentation
- **[Testing Guide](docs/TESTING.md)** - Comprehensive testing documentation (33.7% coverage, 40.5% for testable packages)
- **[Database Migration Guide](docs/DATABASE_MIGRATION_GUIDE.md)** - When to run SQL patches and migrations
- **[Database Security](docs/DATABASE_SECURITY.md)** - Row Level Security (RLS) implementation guide
- **[Code Quality Guidelines](docs/CODE_QUALITY.md)** - Contribution standards

### Project Management
- **[Roadmap](docs/ROADMAP.md)** - Feature roadmap and future plans
- **[Release Process](docs/RELEASE_PROCESS.md)** - Release management guide
- **[Changelog](CHANGELOG.md)** - Version history and changes

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

For enterprise support and custom development:
- Email: valentyn.solomko@gmail.com
- LinkedIn: [valentynsolomko](https://linkedin.com/in/valentynsolomko)
- GitHub: [valpere](https://github.com/valpere)

---

**Built with ❤️ by Valentyn Solomko - Senior Backend Engineering Leader**
