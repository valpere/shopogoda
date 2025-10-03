# Fly.io Deployment Guide

Complete guide for deploying ShoPogoda to Fly.io with free external databases (Supabase + Upstash) - **$0/month forever**.

## Table of Contents

- [Why Fly.io](#why-flyio)
- [Architecture Overview](#architecture-overview)
- [Cost Breakdown](#cost-breakdown)
- [Prerequisites](#prerequisites)
- [Step 1: Supabase PostgreSQL Setup](#step-1-supabase-postgresql-setup)
- [Step 2: Upstash Redis Setup](#step-2-upstash-redis-setup)
- [Step 3: Fly.io Deployment](#step-3-flyio-deployment)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Why Fly.io

**Best for:** Long-term $0 hosting, global deployments, technical users

**Advantages:**
- ✅ **100% FREE forever** with free-tier external services
- ✅ **3 free VMs** (shared CPU, 256MB RAM each)
- ✅ **Global edge network** (fast worldwide)
- ✅ **Docker-native** (works perfectly with ShoPogoda)
- ✅ **No credit card required** for free tier
- ✅ **Auto-scaling** and health checks
- ✅ **Persistent volumes** (3GB free)

**Perfect for:**
- Long-term personal projects
- Global user base (low latency everywhere)
- Learning cloud deployments
- Avoiding monthly costs entirely

## Architecture Overview

### Free-Tier Architecture ($0/month)

```
┌─────────────────────────────────────────────────────────────┐
│                   Fly.io Free Deployment                     │
├─────────────────────────────────────────────────────────────┤
│  Fly.io App (Bot)                                            │
│  ├── 1-2 shared VMs (256MB RAM each)                        │
│  ├── Global edge deployment                                 │
│  ├── HTTPS endpoint for Telegram webhook                    │
│  ├── Auto-scaling based on load                             │
│  └── Health checks at /health                               │
├─────────────────────────────────────────────────────────────┤
│  Supabase PostgreSQL (External - Free Forever)              │
│  ├── 500 MB database storage                                │
│  ├── Unlimited API requests                                 │
│  ├── 50,000 monthly active users                            │
│  ├── 5 GB bandwidth/month                                   │
│  └── Automatic backups                                       │
├─────────────────────────────────────────────────────────────┤
│  Upstash Redis (External - Free Forever)                    │
│  ├── 10,000 commands/day                                    │
│  ├── 256 MB storage                                         │
│  ├── Global edge caching                                    │
│  └── REST API access                                         │
├─────────────────────────────────────────────────────────────┤
│  Supporting Services                                         │
│  ├── Fly.io Container Registry (free)                       │
│  ├── Fly.io Metrics & Logging (free)                        │
│  └── Fly.io Proxy (free)                                    │
└─────────────────────────────────────────────────────────────┘
```

## Cost Breakdown

### Free Tier Limits

**Fly.io (Forever Free):**
- 3 shared-cpu-1x VMs (256MB RAM)
- 160 GB/month outbound data transfer
- 3 GB persistent volume storage

**Supabase (Forever Free):**
- 500 MB PostgreSQL database
- Unlimited API requests
- 50,000 monthly active users
- 5 GB bandwidth/month
- Daily automatic backups

**Upstash (Forever Free):**
- 10,000 Redis commands/day
- 256 MB Redis storage
- Global low-latency access

### Typical Bot Usage

| Resource | Free Limit | Bot Usage | Within Limit? |
|----------|------------|-----------|---------------|
| Fly.io VMs | 3x 256MB | 1-2 VMs | ✅ Yes |
| Fly.io bandwidth | 160 GB/month | ~5-10 GB | ✅ Yes |
| Supabase DB | 500 MB | ~50-100 MB | ✅ Yes |
| Supabase bandwidth | 5 GB/month | ~2-3 GB | ✅ Yes |
| Upstash commands | 10K/day | ~2-5K/day | ✅ Yes |

**Total Monthly Cost: $0**

### When You Might Exceed Free Tier

**Fly.io:**
- Bot serves >10K users/day → Bandwidth exceeds 160GB
- Solution: Optimize caching, compress responses

**Supabase:**
- Store >500MB weather history → Database full
- Solution: Clean old data (30-day retention)

**Upstash:**
- >10,000 Redis commands/day → Rate limited
- Solution: Optimize cache usage, batch operations

**Paid fallback costs:**
- Fly.io: ~$5/month for extra bandwidth
- Supabase: $25/month for Pro tier (2GB DB)
- Upstash: $10/month for 100K commands/day

## Prerequisites

### 1. Get Your API Keys

**Telegram Bot Token:**
```bash
# Message @BotFather on Telegram
/newbot
# Follow prompts, save token
# Format: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz
```

**OpenWeatherMap API Key:**
```bash
# Sign up at https://openweathermap.org/api
# Free tier: 60 calls/minute, 1M calls/month
# Copy API key from dashboard
```

### 2. Install Fly.io CLI

**macOS/Linux:**
```bash
curl -L https://fly.io/install.sh | sh

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.fly/bin:$PATH"

# Reload shell
source ~/.bashrc  # or ~/.zshrc
```

**Windows (PowerShell):**
```powershell
iwr https://fly.io/install.ps1 -useb | iex
```

**Verify installation:**
```bash
flyctl version
```

### 3. Create Fly.io Account

```bash
# Sign up (no credit card required for free tier)
flyctl auth signup

# Or login if you have an account
flyctl auth login
```

## Step 1: Supabase PostgreSQL Setup

### 1.1 Create Supabase Account

```bash
# 1. Go to https://supabase.com
# 2. Click "Start your project"
# 3. Sign in with GitHub (free, no credit card)
# 4. Click "New project"
```

### 1.2 Create Database Project

**In Supabase Dashboard:**
1. Click "New project"
2. Fill in details:
   - **Name**: `shopogoda-db`
   - **Database Password**: Generate strong password (save it!)
   - **Region**: Choose nearest to your users (e.g., `us-east-1`, `eu-central-1`)
   - **Pricing Plan**: Free
3. Click "Create new project"
4. Wait 2-3 minutes for provisioning

### 1.3 Get Connection Details

**In Supabase Dashboard:**
1. Go to **Project Settings** (gear icon)
2. Click **Database** tab
3. Scroll to **Connection string** section
4. Copy **Connection string** (URI format)

**Example connection string:**
```
postgresql://postgres:YOUR_PASSWORD@db.abcdefghijklmnop.supabase.co:5432/postgres
```

**Extract individual values:**
```bash
# Full connection string
DATABASE_URL=postgresql://postgres:YOUR_PASSWORD@db.abcdefghijklmnop.supabase.co:5432/postgres

# Individual components (for compatibility)
DB_HOST=db.abcdefghijklmnop.supabase.co
DB_PORT=5432
DB_NAME=postgres
DB_USER=postgres
DB_PASSWORD=YOUR_PASSWORD
```

### 1.4 Enable Connection Pooling (Recommended)

**Why:** Serverless apps need connection pooling for better performance.

**In Supabase Dashboard:**
1. Go to **Database** → **Connection Pooling**
2. Enable **Transaction Mode** pooling
3. Copy **Connection Pooler** URL (port 6543)

**Use this for production:**
```bash
# Use pooler URL (port 6543) instead of direct connection (port 5432)
DATABASE_URL=postgresql://postgres:YOUR_PASSWORD@db.abcdefghijklmnop.supabase.co:6543/postgres
```

### 1.5 Configure Database (Optional)

**Run migrations via Supabase:**
```bash
# Install Supabase CLI (optional, for migrations)
npm install -g supabase

# Or use SQL Editor in Supabase dashboard
# Go to SQL Editor → New query → Paste schema
```

**Note:** ShoPogoda automatically runs migrations on first start.

## Step 2: Upstash Redis Setup

### 2.1 Create Upstash Account

```bash
# 1. Go to https://upstash.com
# 2. Click "Sign Up" (free, no credit card)
# 3. Sign in with GitHub or email
```

### 2.2 Create Redis Database

**In Upstash Console:**
1. Click "Create Database"
2. Fill in details:
   - **Name**: `shopogoda-cache`
   - **Type**: Regional (cheaper, sufficient for most cases)
   - **Region**: Choose same as Fly.io deployment (e.g., `us-east-1`)
   - **TLS**: Enabled (default, recommended)
3. Click "Create"

### 2.3 Get Connection Details

**In Upstash Console:**
1. Click on `shopogoda-cache` database
2. Scroll to **REST API** section
3. Copy connection details:

```bash
# REST API (recommended for serverless)
UPSTASH_REDIS_REST_URL=https://us1-adapted-bat-12345.upstash.io
UPSTASH_REDIS_REST_TOKEN=AbCdEf123456...

# Or use Redis protocol (traditional)
REDIS_HOST=us1-adapted-bat-12345.upstash.io
REDIS_PORT=6379
REDIS_PASSWORD=AbCdEf123456...
REDIS_URL=redis://default:AbCdEf123456...@us1-adapted-bat-12345.upstash.io:6379
```

**For ShoPogoda, use either:**
- **Option 1**: Individual params (REDIS_HOST, REDIS_PORT, REDIS_PASSWORD)
- **Option 2**: Full URL (REDIS_URL)

### 2.4 Test Connection (Optional)

```bash
# Install Redis CLI
brew install redis  # macOS
sudo apt install redis-tools  # Ubuntu/Debian

# Test connection
redis-cli -h us1-adapted-bat-12345.upstash.io -p 6379 -a YOUR_PASSWORD --tls

# Run test command
> PING
PONG

> SET test "hello"
OK

> GET test
"hello"

> DEL test
(integer) 1

> QUIT
```

## Step 3: Fly.io Deployment

### 3.1 Prepare Application

```bash
# Navigate to project
cd /home/val/wrk/projects/telegram_bot/shopogoda

# Make sure you're on main branch with latest code
git checkout main
git pull
```

### 3.2 Initialize Fly.io App

```bash
# Initialize Fly app
flyctl launch --no-deploy

# Follow prompts:
# - App Name: shopogoda-bot (or your choice)
# - Region: Choose nearest to users (e.g., iad - Ashburn, Virginia)
# - Setup Postgres: No (we're using Supabase)
# - Setup Redis: No (we're using Upstash)
# - Deploy now: No (we'll configure first)
```

**This creates `fly.toml` configuration file.**

### 3.3 Configure fly.toml

**Edit the generated `fly.toml`:**

```toml
# fly.toml app configuration file

app = "shopogoda-bot"
primary_region = "iad"

[build]
  # Use existing Dockerfile
  dockerfile = "Dockerfile"

[env]
  # Application settings
  PORT = "8080"
  LOG_LEVEL = "info"
  BOT_DEBUG = "false"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0
  processes = ["app"]

  [[http_service.checks]]
    grace_period = "10s"
    interval = "30s"
    method = "GET"
    timeout = "5s"
    path = "/health"

[vm]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 256

[[services]]
  protocol = "tcp"
  internal_port = 8080

  [[services.ports]]
    port = 80
    handlers = ["http"]

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

  [[services.tcp_checks]]
    interval = "15s"
    timeout = "2s"
    grace_period = "5s"

[[services.http_checks]]
  interval = "30s"
  grace_period = "10s"
  method = "get"
  path = "/health"
  protocol = "http"
  timeout = "5s"
  tls_skip_verify = false
```

### 3.4 Set Secrets (Environment Variables)

**Set sensitive variables as secrets:**

```bash
# Telegram Bot Token
flyctl secrets set TELEGRAM_BOT_TOKEN="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"

# OpenWeatherMap API Key
flyctl secrets set OPENWEATHER_API_KEY="abcdef1234567890abcdef1234567890"

# Supabase Database (use connection pooler URL)
flyctl secrets set DATABASE_URL="postgresql://postgres:PASSWORD@db.abcdefghijklmnop.supabase.co:6543/postgres"

# Or set individual DB params
flyctl secrets set DB_HOST="db.abcdefghijklmnop.supabase.co"
flyctl secrets set DB_PORT="6543"
flyctl secrets set DB_NAME="postgres"
flyctl secrets set DB_USER="postgres"
flyctl secrets set DB_PASSWORD="YOUR_SUPABASE_PASSWORD"

# Upstash Redis
flyctl secrets set REDIS_URL="redis://default:PASSWORD@us1-adapted-bat-12345.upstash.io:6379"

# Or set individual Redis params
flyctl secrets set REDIS_HOST="us1-adapted-bat-12345.upstash.io"
flyctl secrets set REDIS_PORT="6379"
flyctl secrets set REDIS_PASSWORD="YOUR_UPSTASH_PASSWORD"

# Optional: Slack webhook for notifications
flyctl secrets set SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK"
```

**View configured secrets:**
```bash
flyctl secrets list
```

### 3.5 Deploy Application

```bash
# Deploy to Fly.io
flyctl deploy

# This will:
# 1. Build Docker image
# 2. Push to Fly.io registry
# 3. Deploy to VMs
# 4. Run health checks
# 5. Start serving traffic
```

**Monitor deployment:**
```bash
# Watch deployment logs
flyctl logs

# Check app status
flyctl status

# View app info
flyctl info
```

### 3.6 Get App URL and Set Webhook

```bash
# Get app URL
flyctl info

# Output shows hostname like: shopogoda-bot.fly.dev

# Set Telegram webhook
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -d "url=https://shopogoda-bot.fly.dev/webhook"

# Verify webhook
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
```

### 3.7 Test Your Bot

```bash
# Check health endpoint
curl https://shopogoda-bot.fly.dev/health

# Expected response:
# {"status":"healthy","timestamp":"2025-01-03T..."}

# Test bot on Telegram
# 1. Open Telegram
# 2. Find your bot (@your_bot_username)
# 3. Send: /start
# 4. Try: /weather London
```

✅ **Your bot is now live on Fly.io for $0/month!**

## Configuration

### Environment Variables Reference

**Required (Set as secrets):**
```bash
TELEGRAM_BOT_TOKEN          # From @BotFather
OPENWEATHER_API_KEY         # From openweathermap.org
DATABASE_URL                # Supabase connection string (pooler)
REDIS_URL                   # Upstash connection string
```

**Optional (Set in fly.toml [env] section):**
```bash
PORT                        # 8080 (Fly.io auto-sets)
LOG_LEVEL                   # info, debug, error
BOT_DEBUG                   # true, false
BOT_WEBHOOK_URL             # Auto-set to Fly.io domain
DEMO_MODE                   # true, false (default: false)
```

### Scaling Configuration

**Scale VMs:**
```bash
# Scale to 2 VMs for redundancy
flyctl scale count 2

# Scale back to 1 VM (save resources)
flyctl scale count 1

# Auto-scaling based on load (already configured in fly.toml)
# min_machines_running = 0  # Scales to zero when idle
# auto_start_machines = true
# auto_stop_machines = true
```

**Increase VM resources (costs more):**
```bash
# Upgrade to 512MB RAM (costs ~$3/month)
flyctl scale memory 512

# Upgrade to 1GB RAM (costs ~$6/month)
flyctl scale memory 1024

# Check current scale
flyctl scale show
```

### Custom Domain (Optional)

**Add custom domain:**
```bash
# Add domain
flyctl certs add bot.yourdomain.com

# Fly.io provides instructions for DNS setup
# Add CNAME record: bot.yourdomain.com → shopogoda-bot.fly.dev

# Check certificate status
flyctl certs show bot.yourdomain.com

# Update webhook with custom domain
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook" \
  -d "url=https://bot.yourdomain.com/webhook"
```

## Monitoring

### Fly.io Monitoring

**View metrics:**
```bash
# App status
flyctl status

# Real-time logs
flyctl logs

# Follow logs (live)
flyctl logs -f

# Filter logs by instance
flyctl logs -i <instance-id>

# View metrics in dashboard
flyctl dashboard
```

**Fly.io Dashboard:**
- Go to https://fly.io/dashboard
- Select `shopogoda-bot` app
- View: Metrics, Logs, Machines, Certificates

**Available metrics:**
- Request count
- Response times
- CPU usage
- Memory usage
- Network traffic

### Health Checks

**Manual health check:**
```bash
# Check app health
curl https://shopogoda-bot.fly.dev/health

# Check webhook status
curl "https://api.telegram.org/bot<TOKEN>/getWebhookInfo"
```

**Automatic health checks:**
- Fly.io checks `/health` every 30 seconds
- Auto-restarts unhealthy machines
- Configured in `fly.toml`

### External Database Monitoring

**Supabase Dashboard:**
- Database size: Project Settings → Database
- API requests: Home → API
- Active connections: Database → Connection Pooling

**Upstash Console:**
- Commands/day: Database → Metrics
- Storage used: Database → Overview
- Latency: Database → Metrics

### Alerts

**Fly.io alerts:**
```bash
# No built-in alerting in free tier
# Monitor via dashboard or CLI
```

**Custom monitoring (optional):**
- Use external services: UptimeRobot, Pingdom (free tier)
- Health check URL: `https://shopogoda-bot.fly.dev/health`
- Alert on downtime

## Deployment Workflows

### Update Application

**Deploy new version:**
```bash
# Pull latest code
git pull

# Deploy update
flyctl deploy

# Monitor deployment
flyctl logs -f
```

**Rollback to previous version:**
```bash
# List releases
flyctl releases

# Rollback to previous
flyctl releases rollback

# Or rollback to specific version
flyctl releases rollback <version-number>
```

### Zero-Downtime Deployments

**Fly.io handles this automatically:**
1. Builds new version
2. Starts new machines
3. Waits for health checks to pass
4. Routes traffic to new machines
5. Shuts down old machines

**No action needed - it's automatic!**

### CI/CD with GitHub Actions

**Create `.github/workflows/deploy-flyio.yml`:**

```yaml
name: Deploy to Fly.io

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: superfly/flyctl-actions/setup-flyctl@master

      - name: Deploy to Fly.io
        run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

**Setup:**
1. Get Fly.io API token: `flyctl auth token`
2. Add to GitHub secrets: Settings → Secrets → New secret
3. Name: `FLY_API_TOKEN`, Value: your token
4. Push to main → Auto-deploys

## Troubleshooting

### Common Issues

**1. Deployment fails with "health check timeout"**

```bash
# Check logs for errors
flyctl logs

# Common causes:
# - App not listening on PORT env var
# - Database connection failed
# - Missing environment variables

# Fix: Verify environment variables
flyctl secrets list

# Redeploy
flyctl deploy
```

**2. Database connection errors**

```bash
# Error: "connection refused" or "timeout"

# Check Supabase status
# Visit: https://status.supabase.com

# Verify connection string
flyctl secrets list | grep DATABASE

# Test connection from VM
flyctl ssh console
# Inside VM:
apt update && apt install postgresql-client -y
psql $DATABASE_URL
# Should connect successfully

# Fix: Use connection pooler (port 6543, not 5432)
flyctl secrets set DATABASE_URL="postgresql://postgres:PASSWORD@db.XXX.supabase.co:6543/postgres"
```

**3. Redis connection errors**

```bash
# Error: "connection refused" or "authentication failed"

# Check Upstash status
# Visit: https://status.upstash.com

# Verify Redis URL
flyctl secrets list | grep REDIS

# Test connection
flyctl ssh console
# Inside VM:
apt update && apt install redis-tools -y
redis-cli -u $REDIS_URL ping
# Should return: PONG

# Fix: Verify password and use TLS
flyctl secrets set REDIS_URL="rediss://default:PASSWORD@host.upstash.io:6379"
# Note: rediss:// (with double 's' for TLS)
```

**4. Webhook not receiving updates**

```bash
# Check webhook status
curl "https://api.telegram.org/bot<TOKEN>/getWebhookInfo"

# Common issues:
# - Wrong URL
# - Certificate errors
# - Webhook not set

# Fix: Reset webhook
curl -X POST "https://api.telegram.org/bot<TOKEN>/deleteWebhook"
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook" \
  -d "url=https://shopogoda-bot.fly.dev/webhook"

# Verify
curl "https://api.telegram.org/bot<TOKEN>/getWebhookInfo"
```

**5. App crashes or restarts frequently**

```bash
# Check logs for errors
flyctl logs -f

# Common causes:
# - Out of memory (256MB limit)
# - Database connection leaks
# - Uncaught exceptions

# Fix: Increase memory (costs more)
flyctl scale memory 512

# Or optimize code to use less memory
```

**6. Exceeded Upstash free tier (10K commands/day)**

```bash
# Check usage in Upstash console
# Daily Commands graph shows usage

# Solutions:
# 1. Optimize cache usage (reduce TTL)
# 2. Batch cache operations
# 3. Use local cache for frequent data
# 4. Upgrade to paid plan ($10/month for 100K/day)

# Monitor Redis commands
flyctl logs | grep -i redis
```

**7. Exceeded Supabase free tier (500MB)**

```bash
# Check database size in Supabase dashboard
# Project Settings → Database → Database size

# Solutions:
# 1. Clean old weather data
flyctl ssh console
psql $DATABASE_URL -c "DELETE FROM weather_data WHERE created_at < NOW() - INTERVAL '30 days';"

# 2. Clean old alerts
psql $DATABASE_URL -c "DELETE FROM triggered_alerts WHERE created_at < NOW() - INTERVAL '90 days';"

# 3. VACUUM database
psql $DATABASE_URL -c "VACUUM FULL;"

# 4. Upgrade to Pro ($25/month for 8GB)
```

### Debug Mode

**Enable debug logging:**
```bash
# Set debug mode
flyctl secrets set LOG_LEVEL="debug"
flyctl secrets set BOT_DEBUG="true"

# Restart app
flyctl apps restart

# View detailed logs
flyctl logs -f
```

### SSH Access

**Access running machine:**
```bash
# SSH into VM
flyctl ssh console

# Check environment
env | grep -E 'DB_|REDIS_|BOT_'

# Check processes
ps aux | grep shopogoda

# Check network
curl localhost:8080/health

# Exit
exit
```

## Performance Optimization

### Reduce Cold Starts

**Problem:** Free tier VMs scale to zero when idle.

**Solutions:**
```bash
# Option 1: Keep 1 VM always running (costs ~$3/month)
# Edit fly.toml:
# min_machines_running = 1

# Option 2: Accept cold starts (5-10 second delay)
# No changes needed

# Option 3: Use Telegram bot warmup
# Set up external ping service (UptimeRobot)
# Ping /health every 5 minutes to keep warm
```

### Optimize Database Queries

**Use connection pooling:**
```bash
# Always use Supabase connection pooler (port 6543)
DATABASE_URL=postgresql://postgres:PASSWORD@db.XXX.supabase.co:6543/postgres
```

**Optimize queries in code:**
- Add database indexes (already done)
- Use SELECT only needed columns
- Batch operations where possible

### Optimize Redis Usage

**Stay within 10K commands/day:**
- Increase cache TTL (less frequent updates)
- Batch SET/GET operations
- Use Redis pipelining

**Monitor usage:**
```bash
# Check daily commands in Upstash console
# Metrics → Commands per day
```

### Optimize Docker Image

**Already optimized in Dockerfile:**
- Multi-stage build
- Alpine base image
- Minimal dependencies

**Verify image size:**
```bash
docker images | grep shopogoda
# Should be < 100MB
```

## Cost Management

### Stay Within Free Tier

**Fly.io (3 VMs, 256MB each):**
- ✅ Use 1-2 VMs only
- ✅ Keep auto_stop_machines = true
- ✅ Monitor bandwidth (<160GB/month)

**Supabase (500MB database):**
- ✅ Clean old data monthly
- ✅ Archive instead of storing forever
- ✅ Use efficient data types

**Upstash (10K commands/day):**
- ✅ Optimize cache TTL
- ✅ Batch operations
- ✅ Monitor daily usage

### Monitor Usage

**Fly.io:**
```bash
# Check current usage
flyctl status

# View billing (if exceeded free tier)
flyctl billing
```

**Supabase:**
- Dashboard → Project Settings → Usage
- Database size, API requests, bandwidth

**Upstash:**
- Console → Database → Metrics
- Daily commands, storage used

### Upgrade Paths (If Needed)

**If you exceed free tier:**

| Service | Free Limit | Paid Tier | Cost |
|---------|------------|-----------|------|
| Fly.io | 3 VMs, 160GB | Pay per use | $3-10/month |
| Supabase | 500MB | Pro (8GB) | $25/month |
| Upstash | 10K/day | 100K/day | $10/month |

**Total if exceeded:** $38-45/month

**Better alternatives:**
- Switch to VPS (Hetzner $4/month)
- Use Railway ($5-10/month all-in-one)

## Backup & Migration

### Backup Database

**Automatic backups:**
- Supabase: Daily backups (7-day retention)
- Access: Dashboard → Database → Backups

**Manual backup:**
```bash
# Dump database
flyctl ssh console
pg_dump $DATABASE_URL > /tmp/backup.sql

# Download from VM
flyctl sftp get /tmp/backup.sql ./shopogoda_backup.sql
```

### Restore Database

```bash
# Upload backup to VM
flyctl sftp shell
put ./shopogoda_backup.sql /tmp/backup.sql

# Restore
flyctl ssh console
psql $DATABASE_URL < /tmp/backup.sql
```

### Migrate to Another Platform

**Export everything:**
```bash
# Database dump
flyctl ssh console -C "pg_dump \$DATABASE_URL" > database.sql

# Environment variables
flyctl secrets list > secrets.txt

# Redis data (if needed)
flyctl ssh console -C "redis-cli -u \$REDIS_URL --dump" > redis.rdb
```

**Import to new platform:**
- Restore database with `psql`
- Set environment variables
- Deploy application

## Next Steps

1. ✅ Deploy to Fly.io
2. ✅ Configure Supabase + Upstash
3. ✅ Set Telegram webhook
4. ✅ Test bot functionality
5. 🔄 Monitor usage (stay within free tier)
6. 🔄 Set up GitHub Actions CI/CD
7. 🔄 Configure custom domain (optional)
8. 🔄 Enable external monitoring (UptimeRobot)

## Resources

**Fly.io:**
- [Documentation](https://fly.io/docs)
- [Community Forum](https://community.fly.io)
- [Status Page](https://status.flyio.net)

**Supabase:**
- [Documentation](https://supabase.com/docs)
- [Discord Community](https://discord.supabase.com)
- [Status Page](https://status.supabase.com)

**Upstash:**
- [Documentation](https://docs.upstash.com)
- [Discord Community](https://discord.gg/w9SenAtbme)
- [Status Page](https://status.upstash.com)

**ShoPogoda:**
- [GitHub Repository](https://github.com/valpere/shopogoda)
- [Issues](https://github.com/valpere/shopogoda/issues)

---

**Last Updated**: 2025-01-03
**Maintained by**: [@valpere](https://github.com/valpere)
