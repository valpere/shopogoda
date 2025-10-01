# Deployment Guide

This document provides comprehensive deployment instructions for ShoPogoda across different environments.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Environment Configuration](#environment-configuration)
- [Deployment Methods](#deployment-methods)
- [Health Checks & Monitoring](#health-checks--monitoring)
- [Rollback Procedures](#rollback-procedures)
- [Troubleshooting](#troubleshooting)

## Overview

ShoPogoda supports multiple deployment environments:

- **Development**: Local development with docker-compose
- **Staging**: Pre-production testing environment
- **Production**: Live production environment

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     ShoPogoda Stack                          │
├─────────────────────────────────────────────────────────────┤
│  Bot Application (Go)                                        │
│  ├── Telegram Bot API Integration                           │
│  ├── Weather Service (OpenWeatherMap)                       │
│  └── Health Check Endpoint (:8080/health)                   │
├─────────────────────────────────────────────────────────────┤
│  PostgreSQL 15                                               │
│  ├── User Data & Preferences                                │
│  ├── Weather History                                         │
│  └── Alert Configurations                                    │
├─────────────────────────────────────────────────────────────┤
│  Redis 7                                                     │
│  ├── Weather Data Cache                                      │
│  ├── Session Storage                                         │
│  └── Rate Limiting                                           │
├─────────────────────────────────────────────────────────────┤
│  Observability Stack                                         │
│  ├── Prometheus (Metrics)                                    │
│  ├── Grafana (Dashboards)                                    │
│  └── Jaeger (Distributed Tracing)                           │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

### Required Software

- Docker Engine 20.10+ ([Install](https://docs.docker.com/engine/install/))
- Docker Compose 2.0+ ([Install](https://docs.docker.com/compose/install/))
- Git 2.30+
- Bash 4.0+

### Required Credentials

1. **Telegram Bot Token**: From [@BotFather](https://t.me/BotFather)
2. **OpenWeatherMap API Key**: From [OpenWeatherMap](https://openweathermap.org/api)
3. **(Optional)** Slack/Teams webhook URLs for notifications

### Resource Requirements

| Environment | CPU | Memory | Disk Space |
|-------------|-----|--------|------------|
| Development | 2 cores | 4 GB | 10 GB |
| Staging | 2 cores | 4 GB | 20 GB |
| Production | 4 cores | 8 GB | 50 GB |

## Environment Configuration

### 1. Clone Repository

```bash
git clone https://github.com/valpere/shopogoda.git
cd shopogoda
```

### 2. Configure Environment

#### For Staging:

```bash
cp .env.staging.example .env.staging
vim .env.staging  # Edit with your values
```

#### For Production:

```bash
cp .env.production.example .env.production
vim .env.production  # Edit with your values
```

### 3. Required Environment Variables

**Critical Variables (Must Set):**

```bash
# Bot Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_here
OPENWEATHER_API_KEY=your_api_key_here

# Database Security
DB_PASSWORD=strong_random_password_here

# Production Only
GRAFANA_ADMIN_PASSWORD=another_strong_password
REDIS_PASSWORD=redis_secure_password
```

**Recommended Variables:**

```bash
# Webhook Configuration (Production)
BOT_WEBHOOK_URL=https://your-domain.com/webhook

# Enterprise Integrations
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK
TEAMS_WEBHOOK_URL=https://your-org.webhook.office.com/YOUR/WEBHOOK

# Monitoring
GRAFANA_URL=https://grafana.your-domain.com
```

## Deployment Methods

### Method 1: Automated Deployment Script (Recommended)

#### Deploy to Staging

```bash
./scripts/deploy/deploy.sh staging develop
```

#### Deploy to Production

```bash
./scripts/deploy/deploy.sh production latest
```

#### Deploy Specific Version

```bash
./scripts/deploy/deploy.sh production v1.2.3
```

### Method 2: Manual docker-compose

#### Staging

```bash
docker-compose -f docker/docker-compose.staging.yml up -d
```

#### Production

```bash
docker-compose -f docker/docker-compose.prod.yml up -d
```

### Method 3: CI/CD Pipeline (Automatic)

Deployments are triggered automatically:

- **Staging**: On push to `develop` branch
- **Production**: On push to `main` branch

## Deployment Workflow

The automated deployment script performs these steps:

1. **Prerequisites Check**: Validates Docker, docker-compose, and environment files
2. **Backup Creation**: Saves current deployment state for rollback
3. **Image Pull**: Downloads latest Docker images from registry
4. **Service Start**: Starts all containers with new images
5. **Health Verification**: Waits for all services to become healthy
6. **Deployment Verification**: Tests endpoints and connections
7. **Cleanup**: Removes old backups (keeps last 5)

## Health Checks & Monitoring

### Manual Health Check

```bash
./scripts/deploy/health-check.sh [staging|production]
```

### Service-Specific Health Checks

**Bot Application:**
```bash
curl http://localhost:8080/health
```

**PostgreSQL:**
```bash
docker exec shopogoda-db-prod pg_isready -U shopogoda
```

**Redis:**
```bash
docker exec shopogoda-redis-prod redis-cli ping
```

**Prometheus:**
```bash
curl http://localhost:9090/-/healthy
```

**Grafana:**
```bash
curl http://localhost:3000/api/health
```

### Monitoring Dashboards

After deployment, access:

- **Bot Health**: http://localhost:8080/health
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin / see GRAFANA_ADMIN_PASSWORD)
- **Jaeger**: http://localhost:16686

### Service Logs

**View all logs:**
```bash
docker-compose -f docker/docker-compose.prod.yml logs -f
```

**View specific service:**
```bash
docker-compose -f docker/docker-compose.prod.yml logs -f bot
```

**Last 100 lines:**
```bash
docker-compose -f docker/docker-compose.prod.yml logs --tail=100 bot
```

## Rollback Procedures

### Automatic Rollback

```bash
./scripts/deploy/rollback.sh [staging|production]
```

The script will:
1. List available backups
2. Allow selection (or use latest)
3. Confirm rollback operation
4. Stop current deployment
5. Restore previous configuration
6. Start services with previous version

### Manual Rollback

```bash
# Stop current deployment
docker-compose -f docker/docker-compose.prod.yml down

# Restore backup environment file
cp backups/[timestamp]/.env.production .env.production

# Start with previous configuration
docker-compose -f docker/docker-compose.prod.yml up -d
```

### Backup Management

Backups are stored in `backups/` directory:

```
backups/
├── 20250101_120000/
│   ├── containers.txt
│   ├── images.txt
│   └── .env.production.backup
├── 20250101_140000/
└── ...
```

**List backups:**
```bash
ls -la backups/
```

**Manual backup cleanup:**
```bash
find backups/ -type d -mtime +30 -exec rm -rf {} \;  # Remove backups older than 30 days
```

## Troubleshooting

### Common Issues

#### 1. Container Won't Start

**Symptoms**: Container immediately exits or restarts continuously

**Solutions**:
```bash
# Check container logs
docker logs shopogoda-bot-prod

# Check container status
docker ps -a | grep shopogoda

# Verify environment variables
docker exec shopogoda-bot-prod env
```

#### 2. Database Connection Failed

**Symptoms**: Bot logs show "connection refused" or "authentication failed"

**Solutions**:
```bash
# Check database is running
docker ps | grep postgres

# Test database connection
docker exec shopogoda-db-prod pg_isready

# Verify credentials
docker exec shopogoda-bot-prod env | grep DB_
```

#### 3. Health Check Failures

**Symptoms**: Deployment fails during health check phase

**Solutions**:
```bash
# Check service health status
docker inspect shopogoda-bot-prod | grep -A 10 Health

# Manually test health endpoint
curl -v http://localhost:8080/health

# View recent logs
docker logs --tail=50 shopogoda-bot-prod
```

#### 4. Port Already in Use

**Symptoms**: "port is already allocated" error

**Solutions**:
```bash
# Find process using the port
sudo lsof -i :8080

# Stop conflicting service or change port in .env file
```

#### 5. Out of Disk Space

**Symptoms**: "no space left on device" error

**Solutions**:
```bash
# Check disk usage
df -h

# Clean Docker system
docker system prune -a --volumes

# Remove old images
docker image prune -a
```

### Debug Mode

Enable detailed logging:

```bash
# In .env.staging or .env.production
BOT_DEBUG=true
LOG_LEVEL=debug
```

Then restart services:
```bash
docker-compose -f docker/docker-compose.prod.yml restart bot
```

### Emergency Procedures

#### Complete Service Restart

```bash
docker-compose -f docker/docker-compose.prod.yml restart
```

#### Clean Restart (WARNING: Loses data)

```bash
docker-compose -f docker/docker-compose.prod.yml down -v
docker-compose -f docker/docker-compose.prod.yml up -d
```

#### Force Rebuild

```bash
docker-compose -f docker/docker-compose.prod.yml up -d --force-recreate --build
```

## Security Best Practices

1. **Never commit** `.env` files to version control
2. **Rotate credentials** regularly (every 90 days)
3. **Use strong passwords** for database and Redis
4. **Enable SSL/TLS** in production (`DB_SSL_MODE=require`)
5. **Restrict network access** using firewalls
6. **Monitor logs** for suspicious activity
7. **Keep Docker images updated** (automated via Dependabot)
8. **Use webhook mode** in production (not polling)

## Performance Tuning

### Database Optimization

```bash
# Adjust PostgreSQL settings in docker-compose
environment:
  POSTGRES_MAX_CONNECTIONS: 100
  POSTGRES_SHARED_BUFFERS: 256MB
```

### Redis Optimization

```bash
# Adjust Redis settings
command: redis-server --maxmemory 512mb --maxmemory-policy allkeys-lru
```

### Container Resources

```yaml
# Add to docker-compose service
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 2G
    reservations:
      cpus: '1'
      memory: 1G
```

## Maintenance

### Regular Tasks

**Daily:**
- Monitor Grafana dashboards
- Check health endpoints
- Review error logs

**Weekly:**
- Review backup integrity
- Check disk space usage
- Update dependencies (via Dependabot PRs)

**Monthly:**
- Rotate credentials
- Review and clean old backups
- Performance analysis

### Updates

**Minor Updates:**
```bash
./scripts/deploy/deploy.sh production v1.2.4
```

**Major Updates:**
1. Deploy to staging first
2. Run full test suite
3. Verify all functionality
4. Schedule production deployment
5. Monitor closely after deployment

## Support

For deployment issues:

1. Check this documentation
2. Review logs: `docker-compose logs`
3. Run health check: `./scripts/deploy/health-check.sh`
4. Open issue: https://github.com/valpere/shopogoda/issues

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Reference](https://docs.docker.com/compose/compose-file/)
- [ShoPogoda Architecture](./ARCHITECTURE.md)
- [Development Guide](../README.md)
