# Configuration Guide

This document provides comprehensive information about configuring the ShoPogoda Weather Bot.

## Table of Contents

- [Overview](#overview)
- [Configuration Priority](#configuration-priority)
- [Configuration Files](#configuration-files)
- [Environment Variables](#environment-variables)
- [Configuration Reference](#configuration-reference)
- [Deployment Examples](#deployment-examples)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

ShoPogoda supports multiple configuration methods to accommodate different deployment scenarios:

- **Environment Variables** - Ideal for containers, CI/CD, and sensitive values
- **YAML Configuration Files** - Perfect for non-sensitive settings and documentation
- **.env Files** - Great for local development
- **Built-in Defaults** - Sensible defaults for quick setup

## Configuration Priority

Configuration values are loaded in the following order (highest to lowest precedence):

1. **Environment Variables** (highest priority)
2. **.env file** in current directory
3. **YAML configuration files** (first found):
   - `./shopogoda.yaml` (current directory)
   - `~/.shopogoda.yaml` (home directory)
   - `~/.config/shopogoda.yaml` (user config directory)
   - `/etc/shopogoda.yaml` (system-wide)
4. **Built-in defaults** (lowest priority)

## Configuration Files

### YAML Configuration

Create a `shopogoda.yaml` file in one of the supported locations:

```yaml
# Bot configuration
bot:
  token: ""  # Set via TELEGRAM_BOT_TOKEN env var for security
  debug: false
  webhook_url: ""
  webhook_port: 8080

# Database configuration (SQLite file; no external database)
database:
  path: ./data/shopogoda.db  # Set via DB_PATH env var

# Weather service configuration
weather:
  openweather_api_key: ""  # Set via OPENWEATHER_API_KEY env var
  airquality_api_key: ""   # Set via AIRQUALITY_API_KEY env var
  user_agent: "ShoPogoda-Weather-Bot/1.0 (contact@shopogoda.bot)"

# Logging configuration
logging:
  level: info  # debug, info, warn, error
  format: json  # json, console

# Metrics configuration
metrics:
  port: 2112
  jaeger_endpoint: ""

# Integration configuration
integrations:
  slack_webhook_url: ""
  teams_webhook_url: ""
  grafana_url: "http://localhost:3000"
```

### Environment File (.env)

For local development, copy `.env.example` to `.env`:

```bash
cp .env.example .env
# Edit .env with your actual values
```

**Important:** Never commit `.env` files to version control!

## Environment Variables

All configuration values can be overridden using environment variables:

### Required Variables

```bash
# Telegram Bot Token (REQUIRED)
TELEGRAM_BOT_TOKEN=your_bot_token_here

# OpenWeatherMap API Key (REQUIRED)
OPENWEATHER_API_KEY=your_api_key_here
```

### Optional Variables

```bash
# Bot Settings
BOT_DEBUG=false
BOT_WEBHOOK_URL=
BOT_WEBHOOK_PORT=8080

# Database Settings (SQLite file; parent directory must be writable)
DB_PATH=./data/shopogoda.db

# Weather API Settings
AIRQUALITY_API_KEY=your_api_key
WEATHER_USER_AGENT=ShoPogoda-Weather-Bot/1.0 (contact@example.com)

# Logging Settings
LOG_LEVEL=info
LOG_FORMAT=json

# Monitoring Settings
PROMETHEUS_PORT=2112
JAEGER_ENDPOINT=http://localhost:14268/api/traces

# Integration Settings
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
TEAMS_WEBHOOK_URL=https://your-org.webhook.office.com/...
GRAFANA_URL=http://localhost:3000
```

## Configuration Reference

### Bot Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `token` | string | - | Telegram Bot Token from @BotFather (required) |
| `debug` | bool | `false` | Enable debug logging and verbose output |
| `webhook_url` | string | - | Webhook URL for production (leave empty for polling) |
| `webhook_port` | int | `8080` | Port for webhook server |

### Database Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | string | `./data/shopogoda.db` | Path to the SQLite database file (env: `DB_PATH`). Created if missing; WAL sidecar files `-wal`/`-shm` appear next to it. Single writer: run one instance per file |

### Cache

There is nothing to configure for caching: it is in-process and lost on restart (weather 10m, forecast 1h, air quality 30m, geocoding 24h, user profile 1h, invalidated on updates).

### Weather Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `openweather_api_key` | string | - | OpenWeatherMap API key (required) |
| `airquality_api_key` | string | - | Air quality API key (optional) |
| `user_agent` | string | `ShoPogoda-Weather-Bot/1.0...` | User-Agent for API requests |

### Logging Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | `info` | Log level: debug, info, warn, error |
| `format` | string | `json` | Log format: json, console |

### Metrics Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | `2112` | Metrics port setting (`PROMETHEUS_PORT`); `/metrics` is currently served on the main HTTP port (`webhook_port`) |
| `jaeger_endpoint` | string | - | Jaeger tracing endpoint URL |

### Integrations Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `slack_webhook_url` | string | - | Slack webhook URL for notifications |
| `teams_webhook_url` | string | - | Microsoft Teams webhook URL |
| `grafana_url` | string | `http://localhost:3000` | Grafana dashboard URL |

## Deployment Examples

### Local Development

```bash
# Copy example configuration
cp .env.example .env

# Edit with your values
TELEGRAM_BOT_TOKEN=your_bot_token
OPENWEATHER_API_KEY=your_api_key
BOT_DEBUG=true
LOG_LEVEL=debug

# Run the bot (SQLite file is created automatically)
make run

# Optional: Prometheus, Grafana, Jaeger
make docker-up
```

### Docker Container

```bash
docker run -d --env-file .env.production -p 8080:8080 \
  -v shopogoda_data:/app/data \
  ghcr.io/valpere/shopogoda:latest
```

The image sets `DB_PATH=/app/data/shopogoda.db` and declares `/app/data` as a volume; mount a volume there to persist the database. See [DEPLOYMENT.md](DEPLOYMENT.md).

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: shopogoda
spec:
  template:
    spec:
      containers:
      - name: shopogoda
        image: shopogoda:latest
        env:
        - name: TELEGRAM_BOT_TOKEN
          valueFrom:
            secretKeyRef:
              name: shopogoda-secrets
              key: telegram-token
        - name: OPENWEATHER_API_KEY
          valueFrom:
            secretKeyRef:
              name: shopogoda-secrets
              key: weather-api-key
        - name: DB_PATH
          value: "/app/data/shopogoda.db"
        volumeMounts:
        - name: config
          mountPath: /etc/shopogoda.yaml
          subPath: shopogoda.yaml
      volumes:
      - name: config
        configMap:
          name: shopogoda-config
```

SQLite allows one writer: use `replicas: 1` with `strategy: Recreate` and mount a PersistentVolumeClaim at `/app/data`. Complete manifests are in `deployments/k8s/`.

### Production Server

```bash
# System-wide configuration
sudo mkdir -p /etc
sudo cp shopogoda.yaml /etc/shopogoda.yaml

# Set environment variables
export TELEGRAM_BOT_TOKEN=your_production_token
export OPENWEATHER_API_KEY=your_production_key
export BOT_WEBHOOK_URL=https://yourdomain.com/webhook
export DB_PATH=/var/lib/shopogoda/data/shopogoda.db

# Run with systemd
sudo systemctl start shopogoda
```

## Security Best Practices

### 1. **Sensitive Data Management**

- Never store tokens/passwords in configuration files
- Use environment variables for all sensitive values
- Consider using secret management systems (HashiCorp Vault, AWS Secrets Manager)
- Regularly rotate API keys and tokens

### 2. **Network Security**

- Use webhook mode for production deployments
- Keep the SQLite file on local disk with restrictive permissions (it contains user data)
- Use strong, unique passwords for all services
- Restrict network access with firewalls

### 3. **Configuration Security**

- Set proper file permissions on config files (`chmod 600`)
- Use `.gitignore` to prevent committing sensitive files
- Validate configuration on startup
- Monitor configuration changes

### 4. **Production Checklist**

- [ ] `BOT_DEBUG=false`
- [ ] `LOG_LEVEL=info`
- [ ] `DB_PATH` on a persistent volume, owned by the bot user
- [ ] Webhook URL configured
- [ ] All secrets via environment variables
- [ ] Monitoring and alerting configured
- [ ] Regular SQLite backups enabled (`sqlite3 <file> ".backup out.db"`)

## Troubleshooting

### Configuration Loading Issues

**Problem:** "Config file not found"

```bash
# Check file locations
ls -la ./shopogoda.yaml
ls -la ~/.shopogoda.yaml
ls -la ~/.config/shopogoda.yaml
ls -la /etc/shopogoda.yaml
```

**Problem:** "Environment variables not working"

```bash
# Verify environment variables are set
env | grep -E "(TELEGRAM|DB_|WEATHER_)"

# Check variable names (case-sensitive)
export TELEGRAM_BOT_TOKEN=your_token  # Correct
export telegram_bot_token=your_token  # Wrong
```

### Database Issues

**Problem:** "unable to open database file" / "readonly database"

- Check that the directory of `DB_PATH` exists and is writable by the bot user
- In containers, bind mounts must be chowned to the container user (see [DEPLOYMENT.md](DEPLOYMENT.md))

**Problem:** "database is locked"

- Another process uses the same file; run exactly one instance

### API Key Issues

**Problem:** "Invalid API key"

- Verify key is correct and active
- Check API key permissions and quotas
- Test with curl: `curl "https://api.openweathermap.org/data/2.5/weather?q=London&appid=YOUR_KEY"`

### Logging and Debugging

Enable debug mode to troubleshoot configuration issues:

```bash
export BOT_DEBUG=true
export LOG_LEVEL=debug
./bot
```

Check the logs for configuration loading messages:

```bash
# Look for config loading messages
grep -i "config" logs/shopogoda.log

# Check environment variable parsing
grep -i "env" logs/shopogoda.log
```

## Support

For additional help:

- Check the [main README](../README.md) for general setup instructions
- Review [deployment documentation](../docs/DEPLOYMENT.md)
- Open an issue on GitHub for bugs or feature requests
- Check existing issues for common problems and solutions
