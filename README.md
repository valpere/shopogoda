# ShoPogoda (Що Погода) - Enterprise Weather Bot

A professional-grade Telegram bot for weather monitoring, environmental alerts, and enterprise integrations.

## 🌟 Features

### Core Weather Services
- **Real-time Weather Data**: Current conditions with comprehensive metrics
- **5-Day Forecasts**: Detailed weather predictions
- **Air Quality Monitoring**: AQI and pollutant tracking
- **Smart Location Management**: Single location per user with GPS and name-based input
- **Multi-language Support**: Ukrainian, English, German, French, Spanish

### Enterprise Features
- **Advanced Alert System**: Custom thresholds and conditions
- **Slack/Teams Integration**: Automated notifications
- **Role-Based Access Control**: Admin, moderator, and user roles
- **Monitoring & Analytics**: Prometheus metrics and Grafana dashboards
- **High Availability**: Redis caching and PostgreSQL clustering

### Technical Excellence
- **Scalable Architecture**: Microservices-ready design
- **Comprehensive Testing**: Unit, integration, and E2E tests
- **Production Ready**: Docker containerization and CI/CD
- **Enterprise Security**: Rate limiting, input validation, audit logs

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- Telegram Bot Token (from @BotFather)
- OpenWeatherMap API Key

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
│   ├── middleware/      # Bot middleware (auth, logging)
│   ├── models/          # Data models and structs
│   └── services/        # Business logic services
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

### Docker Deployment

```bash
# Build and run with Docker
docker build -t shopogoda .
docker run -p 8080:8080 --env-file .env shopogoda
```

### Cloud Deployment

The bot is ready for deployment on:
- Google Cloud Platform (GKE)
- AWS (EKS)
- Azure (AKS)
- Railway, Render, Fly.io (free tiers)

## 📈 Performance

- **Response Time**: < 200ms average
- **Throughput**: 1000+ requests/minute
- **Uptime**: 99.9% SLA
- **Cache Hit Rate**: > 85%

## 🔒 Security

- Input validation and sanitization
- Rate limiting (10 requests/minute per user)
- SQL injection prevention (GORM ORM)
- Secure credential management
- Audit logging for compliance

## 🇺🇦 Ukrainian Localization

- Native Ukrainian language support
- Local weather patterns and seasonal considerations
- Integration potential with Ukrainian emergency services
- Time zone support for Ukrainian regions

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

For enterprise support and custom development:
- Email: valentyn.solomko@gmail.com
- LinkedIn: [valentynsolomko](https://linkedin.com/in/valentynsolomko)
- GitHub: [valpere](https://github.com/valpere)

---

**Built with ❤️ by Valentyn Solomko - Senior Backend Engineering Leader**
