# CLAUDE.md - ShoPogoda

## Project Overview

**ShoPogoda** (Що Погода - "What Weather" in Ukrainian) is a production-ready Telegram bot built with Go and gotgbot framework, designed for corporate weather monitoring, environmental compliance, and employee safety alerts. This project demonstrates advanced backend development, enterprise architecture, and DevOps practices suitable for senior-level portfolio showcasing.

## Architecture & Technology Stack

### Core Technologies

- **Language**: Go 1.24+
- **Auxiliary Languages**: Bash, Perl (for scripts and utilities)
- **IDE**: VSCode
- **Bot Framework**: gotgbot v2
- **Database**: PostgreSQL 15+ with GORM ORM
- **Caching**: Redis 7+
- **Web Framework**: Gin (for webhooks and health checks)
- **Configuration**: Viper
- **Logging**: Zerolog (structured JSON logging)

### Infrastructure & Monitoring

- **Containerization**: Docker + Docker Compose
- **Metrics**: Prometheus with custom collectors
- **Visualization**: Grafana dashboards
- **Tracing**: Jaeger for distributed tracing
- **Health Checks**: HTTP endpoints for monitoring

### External Integrations

- **Weather APIs**: OpenWeatherMap (current weather, forecasts, air quality)
- **Enterprise Notifications**: Slack/Teams webhooks
- **Geocoding**: Location resolution and reverse geocoding
- **Rate Limiting**: Token bucket algorithm for API protection

## Business Features

### Core Weather Services

- Real-time weather data with comprehensive metrics (temperature, humidity, pressure, wind, visibility, UV index)
- 5-day weather forecasts with daily min/max temperatures
- Air quality monitoring (AQI, CO, NO₂, O₃, PM2.5, PM10)
- Multi-location management with GPS coordinate support
- Intelligent caching strategy (10min weather, 1hr forecasts, 24hr geocoding)
- Multi-language support (Ukrainian, English, German, French, Spanish)

### Enterprise Alert System

- Custom alert thresholds for temperature, humidity, wind speed, air quality
- Configurable alert conditions (greater than, less than, equal to)
- Severity calculation (Low, Medium, High, Critical)
- Anti-spam protection (1-hour cooldown per alert type)
- Slack/Teams integration for enterprise notifications
- Escalation procedures for critical environmental conditions

### User Management & Security

- Role-based access control (User, Moderator, Admin)
- Rate limiting (10 requests per minute per user)
- Input validation and SQL injection prevention
- Audit logging for compliance requirements
- Session management with Redis
- Multi-tenant architecture support

### Monitoring & Observability

- Prometheus metrics for bot performance, API calls, error rates
- Grafana dashboards for real-time monitoring
- Jaeger distributed tracing for request flow analysis
- Health check endpoints for uptime monitoring
- Structured logging with correlation IDs
- Performance SLA tracking (99.9% uptime, <200ms response time)

## Project Structure

```
shopogoda/
├── cmd/bot/                    # Application entry point
│   └── main.go                 # Main application with graceful shutdown
├── internal/                   # Private application code
│   ├── bot/                    # Bot initialization and HTTP server
│   ├── config/                 # Viper-based configuration management
│   ├── database/               # PostgreSQL and Redis connections
│   ├── handlers/               # Telegram command handlers
│   │   └── commands/           # Weather, location, alert commands
│   ├── middleware/             # Logging, metrics, auth, rate limiting
│   ├── models/                 # GORM models with relationships
│   └── services/               # Business logic services
│       ├── user_service.go     # User management and statistics
│       ├── weather_service.go  # Weather API integration
│       ├── location_service.go # Location management
│       ├── alert_service.go    # Alert processing engine
│       ├── notification_service.go # Slack/Teams integration
│       └── scheduler_service.go # Background job processing
├── pkg/                        # Public libraries
│   ├── weather/                # Weather API clients
│   ├── alerts/                 # Alert engine utilities
│   └── metrics/                # Prometheus metrics collectors
├── deployments/                # Infrastructure as Code
│   ├── docker-compose.yml      # Development environment
│   ├── Dockerfile              # Production container
│   ├── prometheus.yml          # Metrics configuration
│   └── grafana/                # Dashboard configurations
├── scripts/                    # Build and deployment scripts
│   ├── migrate.go              # Database migration runner
│   └── init.sql                # Database initialization
├── tests/                      # Test suites
│   ├── integration/            # Integration tests
│   └── e2e/                    # End-to-end tests
├── Makefile                    # Build automation
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
├── .env.example                # Environment configuration template
├── README.md                   # Project documentation
└── CLAUDE.md                   # This file - AI development context
```

## Development Workflow

### Local Development Setup

```bash
# Clone and initialize ShoPogoda project
git clone <repository>
cd shopogoda
make init

# Configure environment variables
cp .env.example .env
# Edit .env with API keys (TELEGRAM_BOT_TOKEN, OPENWEATHER_API_KEY)

# Start development infrastructure
make docker-up

# Build and run ShoPogoda application
make build
make migrate
make run
```

### Testing Strategy

```bash
# Run all tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run end-to-end tests
make test-e2e

# Performance testing
make test-performance

# Lint code quality
make lint
```

### Deployment Options

- **Local Development**: Docker Compose with hot reload
- **Free Cloud Hosting**: Railway.app, Render.com, Fly.io
- **Enterprise Cloud**: GCP Cloud Run, AWS ECS, Azure Container Instances
- **Kubernetes**: Helm charts for production deployment

## Enterprise Integration Capabilities

### Slack/Teams Integration

- Automated weather alerts with severity-based formatting
- Daily weather summaries for team channels
- Interactive buttons for weather queries
- Escalation procedures for critical alerts
- Custom webhook configurations per organization

### Role-Based Access Control

- **Admin**: System statistics, user management, broadcast messages
- **Moderator**: Alert configuration, location management
- **User**: Weather queries, personal alerts, location settings

### Compliance & Reporting

- Audit trail for all user actions
- Environmental compliance monitoring
- Automated reporting for safety teams
- Data export capabilities (JSON, CSV, PDF)
- GDPR compliance for EU operations

### Monitoring & Alerting

- SLA monitoring (99.9% uptime target)
- Performance metrics (response time < 200ms)
- Error rate tracking and alerting
- Resource usage monitoring
- Custom dashboards for stakeholders

## API Design & Documentation

### Telegram Bot Commands
```
/start          - Welcome message with feature overview (Ukrainian/English)
/weather [loc]  - Current weather for location (Поточна погода)
/forecast [loc] - 5-day weather forecast (Прогноз на 5 днів)
/air [loc]      - Air quality information (Якість повітря)
/addlocation    - Add monitoring location (Додати локацію)
/subscribe      - Set up notifications (Підписатися)
/addalert       - Create weather alerts (Створити сповіщення)
/settings       - User preferences (Налаштування)
/help           - Complete command reference (Допомога)

# Admin commands
/stats          - System statistics
/broadcast      - Send message to all users
/users          - User management
```

### HTTP Endpoints
```
GET  /health     - Health check endpoint
GET  /metrics    - Prometheus metrics
POST /webhook    - Telegram webhook receiver
GET  /status     - Detailed system status
GET  /version    - Application version info
```

### Database Schema

- **Users**: Authentication, preferences, language settings
- **Locations**: Geographic monitoring points with timezone support
- **WeatherData**: Historical weather records with indexing
- **AlertConfigs**: User-defined alert rules with conditions
- **EnvironmentalAlerts**: Triggered alerts log with severity
- **Subscriptions**: Notification preferences and scheduling
- **UserSessions**: Session management and state tracking

## Performance & Scalability

### Performance Targets

- **Response Time**: < 200ms average for weather queries
- **Throughput**: 1000+ concurrent users
- **Cache Hit Rate**: > 85% for weather data
- **Uptime**: 99.9% SLA with health checks
- **API Rate Limits**: Respect OpenWeatherMap limits (1000 calls/day free tier)

### Scalability Features

- Horizontal scaling with load balancing
- Database connection pooling (25 connections)
- Redis clustering for high availability
- Stateless application design
- Graceful shutdown handling
- Auto-scaling based on metrics

### Resource Optimization

- Efficient API rate limiting
- Intelligent data caching strategies
- Database query optimization with indexes
- Memory usage monitoring and limits
- Container resource constraints
- Background job processing with queues

## Security Implementation

### Input Validation

- SQL injection prevention with GORM parameterized queries
- Command injection protection for user inputs
- Rate limiting per user (10 requests/minute)
- Input sanitization for all user data
- XSS prevention for web interfaces

### Authentication & Authorization

- Telegram user authentication via bot API
- Role-based command access control
- Session management with Redis TTL
- API key security for external services
- JWT tokens for webhook authentication

### Data Protection

- Environment-based configuration management
- Secure credential storage (no hardcoded secrets)
- Audit logging for sensitive operations
- Data encryption for PII compliance
- GDPR compliance for EU users

## Quality Assurance

### Code Style

- Use Go standard formatting (`gofmt`)
- Follow effective Go patterns
- Write clear, self-documenting code
- Add comments for complex logic
- Use meaningful variable names (short names for loops: `i`, `j`, `e` for events)

### Code Quality

- Go best practices and idioms
- Comprehensive error handling with context
- Structured logging throughout application
- Code coverage > 80% target
- Linting with golangci-lint
- Code review process with GitHub Actions

### Testing Strategy

- Unit tests for all services and handlers
- Integration tests for database operations
- End-to-end tests for user workflows
- Performance testing for high load scenarios
- Security testing for vulnerabilities
- Chaos engineering for reliability testing

### Documentation

- Comprehensive README with setup guide
- API documentation with examples
- Architecture diagrams and flowcharts
- Deployment guides for multiple platforms
- Troubleshooting and support documentation
- Inline code documentation with godoc

#### Sites With Documentation

- Main site: https://valpere.github.io/shopogoda/
- API docs: https://valpere.github.io/shopogoda/docs/
- GitHub: https://github.com/valpere/shopogoda

## Business Value & Use Cases

### Corporate Facility Management

- Monitor environmental conditions across multiple locations in Ukraine and internationally
- Automated alerts for extreme weather conditions affecting operations
- Compliance reporting for workplace safety regulations
- Integration with existing enterprise systems and workflows

### Employee Safety & Communication

- Proactive weather warnings for field workers and remote teams
- Air quality alerts for sensitive employees and health conditions
- Emergency notification system for severe weather events
- Mobile-first interface for field access and real-time updates

### Cost Savings & Efficiency

- Reduce manual weather monitoring overhead
- Prevent equipment damage from extreme weather conditions
- Optimize resource allocation based on weather predictions
- Automate compliance reporting processes for regulatory bodies

### Ukrainian Market Specific Features

- Native Ukrainian language support (Що Погода)
- Local weather patterns and seasonal considerations
- Integration with Ukrainian emergency services
- Time zone support for multiple Ukrainian regions

## Development Context for AI Assistance

### Code Generation Guidelines

- Follow Go best practices and idiomatic patterns
- Use dependency injection for service layers
- Implement comprehensive error handling with wrapped errors
- Maintain consistency with existing code style and patterns
- Add appropriate logging and metrics to new features

### Architecture Patterns

- Clean architecture with clear separation of concerns
- Repository pattern for data access layers
- Service layer for business logic implementation
- Middleware pattern for cross-cutting concerns
- Event-driven architecture for notifications

### Testing Requirements

- Write unit tests for all new functions and methods
- Include integration tests for database operations
- Add end-to-end tests for new user workflows
- Maintain code coverage above 80%
- Include performance benchmarks for critical paths

### Documentation Standards

- Add godoc comments for all exported functions
- Update README.md for new features or setup changes
- Include examples in API documentation
- Update architecture diagrams for structural changes
- Maintain changelog for version releases

## Future Enhancements

### Advanced Features

- Machine learning for weather pattern prediction
- IoT sensor integration for local monitoring
- Mobile app development (React Native)
- Advanced analytics and reporting dashboard
- Voice interface support for accessibility

### Enterprise Integrations

- LDAP/Active Directory authentication
- Salesforce CRM integration
- Microsoft Teams native app
- ServiceNow incident management
- SAP ERP integration for resource planning

### Scalability Improvements

- Microservices architecture migration
- Event-driven architecture with message queues
- Multi-region deployment for global coverage
- Advanced caching with CDN integration
- Kubernetes operator for automated management

### Ukrainian Localization

- Integration with Ukrainian Hydrometeorological Service
- Support for Ukrainian administrative divisions
- Local emergency alert system integration
- Cultural and seasonal event notifications
- Ukrainian language NLP for voice commands

This ShoPogoda project demonstrates enterprise-level software development capabilities, combining modern Go development practices with production-ready infrastructure and comprehensive business functionality tailored for Ukrainian and international markets.
