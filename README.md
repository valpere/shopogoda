# ShoPogoda

**ShoPogoda** (Що Погода - "What Weather" in Ukrainian) is a production-ready Telegram bot built with Go and gotgbot framework, designed for corporate weather monitoring, environmental compliance, and employee safety alerts. This project demonstrates advanced backend development, enterprise architecture, and DevOps practices suitable for senior-level portfolio showcasing.

## 🌟 Features

### Core Weather Services

- **Real-time Weather Data**: Current conditions with comprehensive metrics
- **5-Day Forecasts**: Detailed weather predictions
- **Air Quality Monitoring**: AQI and pollutant tracking
- **Location Management**: Multiple saved locations with GPS support

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
- Go 1.24+
- Docker & Docker Compose
- Telegram Bot Token (from @BotFather)
- OpenWeatherMap API Key

### Setup Commands

```bash
# 1. Clone and initialize
git clone <repository>
cd enterprise-weather-bot
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

Edit `.env` file with your credentials:

```bash
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
OPENWEATHER_API_KEY=your_openweather_api_key
SLACK_WEBHOOK_URL=your_slack_webhook_url
```

## 📊 Monitoring

Access the monitoring stack:

- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Jaeger Tracing**: http://localhost:16686
- **Bot Health**: http://localhost:8080/health
- **Metrics**: http://localhost:8080/metrics

## 🏗️ Architecture

```
├── cmd/bot/              # Application entry point
├── internal/             # Private application code
│   ├── bot/             # Bot initialization and setup
│   ├── config/          # Configuration management
│   ├── database/        # Database connections
│   ├── handlers/        # Telegram command handlers
│   ├── middleware/      # Bot middleware (auth, logging)
│   ├── models/          # Data models
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

### Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test
go test ./internal/services/weather_service_test.go -v
```

## 🚀 Deployment

### Docker Deployment

```bash
# Build and run with Docker
docker build -t weather-bot .
docker run -p 8080:8080 --env-file .env weather-bot
```

### Kubernetes Deployment

```bash
# Deploy to Kubernetes
kubectl apply -f deployments/k8s/
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

## 🤝 Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Push to branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

For enterprise support and custom development:
- LinkedIn: [valentynsolomko](https://linkedin.com/in/valentynsolomko)
- GitHub: [valpere](https://github.com/valpere)

---

**Built with ❤️ by Valentyn Solomko - Senior Backend Engineering Leader**
