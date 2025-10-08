# Railway Deployment Guide

Complete guide for deploying ShoPogoda to Railway - simple container platform with excellent developer experience.

## Table of Contents

- [Why Railway](#why-railway)
- [Cost Breakdown](#cost-breakdown)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
  - [Method 1: Web Dashboard (Integrated)](#method-1-web-dashboard-recommended---10-minutes)
  - [Method 2: Railway CLI](#method-2-railway-cli-advanced)
- [Deployment Variant 2: Railway + Supabase + Upstash](#deployment-variant-2-railway--supabase--upstash)
  - [Step 1: Create Supabase Database](#step-1-create-supabase-database)
  - [Step 2: Create Upstash Redis](#step-2-create-upstash-redis)
  - [Step 3: Deploy Bot to Railway](#step-3-deploy-bot-to-railway)
  - [Step 4: Verify Deployment](#step-4-verify-deployment)
  - [Step 5: Monitor Free Tier Usage](#step-5-monitor-free-tier-usage)
  - [Troubleshooting Hybrid Setup](#troubleshooting-hybrid-setup)
  - [Cost Comparison](#cost-comparison-integrated-vs-hybrid)
  - [Migration Between Variants](#migration-between-variants)
- [Detailed Setup](#detailed-setup)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Migration & Backup](#migration--backup)

## Deployment Options

This guide covers **two deployment variants**:

| Variant | Cost | Complexity | Best For |
|---------|------|------------|----------|
| **Variant 1: Integrated** | $8-14/month | ⭐ Simple | Production, Always-on, Low latency |
| **Variant 2: Hybrid** | $0/month | ⭐⭐⭐ Moderate | Testing, Budget-conscious, Small bots |

- **Variant 1 (Integrated)**: Railway hosts bot + PostgreSQL + Redis (all in one platform)
- **Variant 2 (Hybrid)**: Railway hosts bot only, uses Supabase (PostgreSQL) + Upstash (Redis)

**Quick Decision:**
- 💰 Have $8-14/month budget? → Use **Variant 1** (simpler, faster)
- 🆓 Want completely free? → Use **Variant 2** (more setup, free tier limits)

## Why Railway

**Best for:** Rapid development, MVPs, full-stack apps with database

**Advantages:**
- ✅ **500 instance hours/month free** (1 GB RAM included)
- ✅ **One-click deployment** from GitHub
- ✅ **PostgreSQL & Redis included** in platform
- ✅ **Automatic Dockerfile detection** - zero config needed
- ✅ **Automatic HTTPS** with free subdomain
- ✅ **Built-in monitoring** and real-time logs
- ✅ **Usage-based pricing** - pay only for what you use

**Perfect for:**
- Personal projects and MVPs
- Development/testing environments
- Small to medium bots (up to 50K users)
- Teams needing quick deployments
- Projects requiring always-on database

**Note:** Railway changed their pricing in 2023. Free tier now includes 500 instance hours/month with 1 GB RAM. Paid plans start at usage-based billing (~$5-10/month for typical bot usage).

## Cost Breakdown

### Free Tier (Current 2025)

**Monthly Allowance:**
- **500 instance hours/month** (enough for ~20 days always-on or full month with sleep)
- **1 GB RAM** per instance
- Includes: compute, bandwidth, metrics
- **Database & Redis**: Billed separately (see below)

**Typical ShoPogoda Bot Usage:**

| Resource | Configuration | Hours/Month | Cost Estimate | Notes |
|----------|--------------|-------------|---------------|-------|
| **Bot Service** | 1 GB RAM | 500 (free tier) | **$0** | Within free hours |
| **PostgreSQL 15** | 512 MB RAM, 1 GB storage | 730 (always-on) | **$5** | Shared instance |
| **Redis 7** | 256 MB RAM | 730 (always-on) | **$3** | Cache only |
| **Bandwidth** | ~10 GB/month | Included | **$0** | Free tier covers it |
| **Total** | - | - | **~$8/month** | Predictable cost |

**Free Tier Strategy:**
- ✅ Bot uses all 500 free hours → $0 for bot
- ✅ Database + Redis always-on → ~$8/month combined
- ✅ No unexpected costs from API calls or bandwidth

### Paid Usage (After Free Hours)

If your bot needs more than 500 hours/month (always-on = 730 hours):

**Usage-Based Pricing:**
- Compute: **$0.000231/GB-second** ($6/GB/month for always-on)
- Example: 1 GB instance always-on = ~$6/month
- **Total with database:** ~$14/month ($6 bot + $5 DB + $3 Redis)

**Cost Optimization Tips:**
1. **Efficient caching** → Reduce external API calls (OpenWeatherMap)
2. **Database cleanup** → Delete old weather data (>30 days)
3. **Connection pooling** → Minimize database connections (max 5-10)
4. **Health check optimization** → Reduce frequency if high traffic

### Comparison with Alternatives

| Platform | Bot | Database | Redis | Total/Month |
|----------|-----|----------|-------|-------------|
| **Railway** | $0-6 | $5 | $3 | **$8-14** |
| Fly.io | $5-7 | $0 (Supabase) | $0 (Upstash) | $5-7 |
| Vercel | $0 | $0 (Supabase) | $0 (Upstash) | $0 (cold starts) |
| Replit | $20 | Included | No Redis | $20 |
| GCP Cloud Run | $5-10 | $15-20 | $10 | $30-40 |

**Railway is best when:**
- You need always-on bot (no cold starts)
- You want integrated database without external services
- You prefer predictable monthly costs
- Development experience matters (excellent CLI/dashboard)

## Prerequisites

Before deploying to Railway, ensure you have:

**Required:**
1. ✅ **GitHub account** (for Railway authentication and deployment)
2. ✅ **Telegram Bot Token** from [@BotFather](https://t.me/BotFather)
3. ✅ **OpenWeatherMap API Key** from [openweathermap.org](https://openweathermap.org/api)

**Optional:**
- Railway CLI (for local deployment and debugging)
- Docker installed (for local testing before deployment)
- Git CLI (for repository management)

**Project Requirements:**
- Dockerfile located at `docker/Dockerfile` ✅ (ShoPogoda has this)
- Health check endpoint at `/health` ✅ (implemented)
- Port 8080 exposed ✅ (default in ShoPogoda)

## Quick Start

### Method 1: Web Dashboard (Recommended - 10 Minutes)

**Step 1: Create Railway Account**
```
1. Go to https://railway.app
2. Click "Login with GitHub"
3. Authorize Railway to access your repositories
4. Accept terms of service
```

**Step 2: Create New Project from GitHub**
```
1. Click "New Project" button
2. Select "Deploy from GitHub repo"
3. Choose your fork of shopogoda (or valpere/shopogoda)
4. Railway detects Dockerfile at docker/Dockerfile
5. Click "Deploy Now"
```

**Note:** If Railway doesn't detect the Dockerfile, add a `railway.toml` file (see [Detailed Setup](#railway-configuration-file)).

**Step 3: Add Database Services**
```
1. In your project dashboard, click "+ New"
2. Select "Database" → "Add PostgreSQL"
3. Wait for PostgreSQL to provision (~30 seconds)
4. Click "+ New" again
5. Select "Database" → "Add Redis"
6. Wait for Redis to provision (~30 seconds)
```

**Step 4: Configure Environment Variables**

Click on your **bot service** (not database), then go to **Variables** tab:

**Required Variables:**
```bash
# Telegram Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
OPENWEATHER_API_KEY=your_api_key_from_openweathermap

# Bot Mode (webhook for Railway)
BOT_WEBHOOK_MODE=true
BOT_WEBHOOK_URL=${{RAILWAY_PUBLIC_DOMAIN}}
```

**Database Variables (Auto-configured by Railway):**
```bash
# PostgreSQL - use Railway's service variables
DATABASE_URL=${{Postgres.DATABASE_URL}}
DB_HOST=${{Postgres.PGHOST}}
DB_PORT=${{Postgres.PGPORT}}
DB_NAME=${{Postgres.PGDATABASE}}
DB_USER=${{Postgres.PGUSER}}
DB_PASSWORD=${{Postgres.PGPASSWORD}}
DB_SSL_MODE=disable

# Redis - use Railway's service variables
REDIS_URL=${{Redis.REDIS_URL}}
REDIS_HOST=${{Redis.REDIS_HOST}}
REDIS_PORT=${{Redis.REDIS_PORT}}
REDIS_PASSWORD=${{Redis.REDIS_PASSWORD}}
REDIS_DB=0
```

**Optional Variables:**
```bash
LOG_LEVEL=info
LOG_FORMAT=json
BOT_DEBUG=false
DEMO_MODE=false
```

**Step 5: Generate Public Domain**
```
1. Go to your bot service → Settings
2. Scroll to "Networking" section
3. Click "Generate Domain"
4. Copy the generated domain (e.g., shopogoda-production.up.railway.app)
5. Add it to BOT_WEBHOOK_URL variable (or use RAILWAY_PUBLIC_DOMAIN)
```

**Step 6: Deploy and Monitor**
```
1. Railway automatically triggers deployment
2. Go to "Deployments" tab to watch build progress
3. Build time: ~3-5 minutes (Go compilation + Docker layers)
4. Watch logs in real-time (click on deployment)
5. Wait for "Deployment successful" message
```

**Step 7: Verify Webhook**

Once deployed, verify Telegram webhook is set correctly:

```bash
# Check webhook status
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"

# Expected output should show:
# "url": "https://your-app.up.railway.app/webhook"
# "has_custom_certificate": false
# "pending_update_count": 0
```

If webhook is not set automatically, set it manually:

```bash
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://your-app.up.railway.app/webhook"}'
```

**Step 8: Test Your Bot**
```
1. Open Telegram and find your bot
2. Send: /start
   Expected: Welcome message with instructions
3. Send: /weather London
   Expected: Current weather for London
4. Send: /forecast Kyiv
   Expected: 5-day weather forecast
```

✅ **Deployment Complete!** Your bot is now live on Railway.

**Troubleshooting First Deployment:**
- ❌ "Dockerfile not found" → Add railway.toml with correct path
- ❌ "Database connection failed" → Check DB_ variables are using ${{Postgres.*}}
- ❌ "Bot not responding" → Check logs for errors, verify webhook URL
- ❌ "Build timeout" → Free tier might be slow, wait or upgrade

### Method 2: Railway CLI (Advanced)

**1. Install Railway CLI**

```bash
# macOS/Linux (recommended)
curl -fsSL https://railway.app/install.sh | sh

# Windows (PowerShell)
iwr https://railway.app/install.ps1 | iex

# Verify installation
railway --version
# Expected: railway 4.x.x or higher
```

**Note:** Do NOT use `npm install -g @railway/cli` - it may have permission issues. Use the official installer above.

**2. Login to Railway**

```bash
railway login
```

**What happens:**
- Opens browser for authentication
- Login with your email
- Check email for login code
- Enter code in browser
- CLI confirms: "Logged in as [your-email]"

**3. Initialize Project and Deploy**

```bash
cd /path/to/shopogoda

# Deploy from GitHub (creates service automatically)
railway up
```

**What happens:**
- Prompts to link GitHub repository (if not linked)
- Creates a service in Railway project
- Uploads code and builds Docker image
- First deployment will start

**Alternative: Link to existing project**

```bash
# If you already created project via web dashboard
railway link

# Then deploy
railway up
```

**4. Add Services (For Variant 1 - Integrated Only)**

**Skip this step for Variant 2 (Hybrid)** - you're using external Supabase + Upstash.

For Variant 1 (integrated Railway databases):
```bash
# Add PostgreSQL
railway add --database postgres

# Add Redis
railway add --database redis
```

**5. Set Environment Variables**

**Railway CLI v4+ uses `--set` flag (not `set` command):**

```bash
# Required variables
railway variables --set TELEGRAM_BOT_TOKEN="your_token_here"
railway variables --set OPENWEATHER_API_KEY="your_api_key_here"
railway variables --set BOT_WEBHOOK_MODE="true"

# Database variables (Variant 1 - Integrated)
railway variables --set DATABASE_URL='${{Postgres.DATABASE_URL}}'
railway variables --set DB_HOST='${{Postgres.PGHOST}}'
railway variables --set DB_PORT='${{Postgres.PGPORT}}'
railway variables --set DB_NAME='${{Postgres.PGDATABASE}}'
railway variables --set DB_USER='${{Postgres.PGUSER}}'
railway variables --set DB_PASSWORD='${{Postgres.PGPASSWORD}}'
railway variables --set DB_SSL_MODE="disable"

# Redis variables (Variant 1 - Integrated)
railway variables --set REDIS_URL='${{Redis.REDIS_URL}}'
railway variables --set REDIS_HOST='${{Redis.REDIS_HOST}}'
railway variables --set REDIS_PORT='${{Redis.REDIS_PORT}}'
railway variables --set REDIS_PASSWORD='${{Redis.REDIS_PASSWORD}}'
railway variables --set REDIS_DB="0"

# Optional settings
railway variables --set LOG_LEVEL="info"
railway variables --set LOG_FORMAT="json"
railway variables --set BOT_DEBUG="false"
```

**For Variant 2 (Hybrid - Supabase + Upstash):**
See [Variant 2 CLI Setup](#variant-2-cli-deployment) below.

**6. Generate Public Domain**

```bash
railway domain
```

**What happens:**
- Railway generates domain: `your-app-production.up.railway.app`
- Copy this domain for webhook configuration

**7. Set Webhook URL**

```bash
# Replace YOUR_DOMAIN with actual domain from step 6
railway variables --set BOT_WEBHOOK_URL="https://YOUR_DOMAIN.up.railway.app"
```

**8. Redeploy (to apply new variables)**

```bash
railway up
```

**9. Monitor Deployment**

```bash
# View real-time logs
railway logs --follow

# Look for:
# ✅ "Database connected successfully"
# ✅ "Redis connected successfully"
# ✅ "Bot started successfully"
```

**10. Set Telegram Webhook**

```bash
# Replace YOUR_BOT_TOKEN and YOUR_DOMAIN
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://YOUR_DOMAIN.up.railway.app/webhook"}'
```

**11. Verify Deployment**

```bash
# Check webhook status
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"

# Test health endpoint
curl "https://YOUR_DOMAIN.up.railway.app/health"

# Expected: {"status":"healthy","timestamp":"..."}
```

---

### Variant 2 CLI Deployment

**Step-by-step CLI deployment for Variant 2 (Railway + Supabase + Upstash):**

**1. Deploy to Railway**

```bash
cd /path/to/shopogoda
railway login
railway up
```

**2. Set Supabase + Upstash Variables**

```bash
# Telegram & Weather API
railway variables --set TELEGRAM_BOT_TOKEN="your_telegram_bot_token"
railway variables --set OPENWEATHER_API_KEY="your_openweather_api_key"

# Bot webhook mode
railway variables --set BOT_WEBHOOK_MODE="true"

# Supabase PostgreSQL (use connection pooler - port 6543!)
railway variables --set DATABASE_URL="postgresql://postgres.PROJECT:[PASSWORD]@aws-X-REGION.pooler.supabase.com:6543/postgres"
railway variables --set DB_HOST="aws-X-REGION.pooler.supabase.com"
railway variables --set DB_PORT="6543"
railway variables --set DB_NAME="postgres"
railway variables --set DB_USER="postgres.PROJECT"
railway variables --set DB_PASSWORD="your_supabase_password"
railway variables --set DB_SSL_MODE="require"

# Upstash Redis (REST API recommended)
railway variables --set UPSTASH_REDIS_REST_URL="https://xxxxx.upstash.io"
railway variables --set UPSTASH_REDIS_REST_TOKEN="your_upstash_token"

# Optional
railway variables --set LOG_LEVEL="info"
railway variables --set LOG_FORMAT="json"
railway variables --set BOT_DEBUG="false"
```

**3. Generate domain and set webhook URL**

```bash
# Generate domain
railway domain

# Set webhook URL (replace YOUR_DOMAIN)
railway variables --set BOT_WEBHOOK_URL="https://YOUR_DOMAIN.up.railway.app"

# Redeploy with new variables
railway up
```

**4. Monitor and verify**

```bash
# View logs
railway logs --follow

# Set Telegram webhook
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://YOUR_DOMAIN.up.railway.app/webhook"}'
```

---

### CLI Troubleshooting

**Issue: "No service linked"**

```bash
# List services in project
railway service

# Link to a specific service
railway service your-service-name

# Or let Railway detect from current directory
railway link
```

**Issue: "unexpected argument 'set' found"**

Railway CLI v4+ syntax changed. Use:
```bash
railway variables --set KEY="value"  # Correct ✅
# NOT: railway variables set KEY="value"  # Old syntax ❌
```

**Issue: Variables not applying**

After setting variables, redeploy:
```bash
railway up
```

**Issue: Permission denied during npm install**

Don't use npm. Use official installer:
```bash
curl -fsSL https://railway.app/install.sh | sh
```

**Issue: Can't connect to GitHub repo**

If web dashboard hangs retrieving repos, use CLI:
```bash
railway up  # Will prompt for GitHub repo selection
```

## Deployment Variant 2: Railway + Supabase + Upstash

**Total Cost: $0/month** | **Setup Time: 20 minutes** | **Status: ✅ Tested & Working (2025-01-08)**

This variant uses Railway for bot hosting with free external database services to eliminate all costs.

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Railway (Bot Only)                                      │
│  └── ShoPogoda Bot (1 GB RAM, 500 hours/month free)    │
│      ✅ Automatic TLS for Redis                         │
│      ✅ Prepared statements disabled for Supabase       │
│      ✅ Environment-only config (no YAML)               │
└──────────────────┬──────────────────────────────────────┘
                   │
        ┌──────────┴──────────┐
        │                     │
        ▼                     ▼
┌───────────────┐     ┌──────────────┐
│ Supabase      │     │ Upstash      │
│ PostgreSQL 15 │     │ Redis 7      │
│ 500MB free    │     │ 10K cmd/day  │
│ Port 6543     │     │ TLS enabled  │
└───────────────┘     └──────────────┘
```

### Recent Improvements (v1.0.2+)

ShoPogoda includes automatic fixes for common Railway + Supabase + Upstash deployment issues:

1. ✅ **YAML Config Disabled**: All configuration via environment variables only (no config file parsing)
2. ✅ **Automatic Redis TLS**: TLS automatically enabled for cloud Redis hosts (Upstash)
3. ✅ **Supabase Pooler Support**: Prepared statements disabled for transaction pooler compatibility
4. ✅ **Migration Strategy**: Database migrations disabled for pooler, tables created on first run

**No manual workarounds needed** - just set environment variables and deploy!

### Step 1: Create Supabase Database

**1.1 Sign up for Supabase**
```
1. Go to https://supabase.com
2. Sign in with GitHub
3. Create new organization (or use existing)
```

**1.2 Create new project**
```
1. Click "New Project"
2. Name: shopogoda-db
3. Database Password: Generate strong password (save it!)
4. Region: Choose closest to your users
5. Click "Create new project"
6. Wait 2-3 minutes for provisioning
```

**1.3 Get connection details**
```
1. Go to Project Settings → Database
2. Copy these values:
   - Host: db.xxxxxxxxxxxxx.supabase.co
   - Database name: postgres
   - Port: 5432
   - User: postgres
   - Password: [your generated password]

3. Or copy the connection string:
   postgresql://postgres:[PASSWORD]@db.xxxxx.supabase.co:5432/postgres
```

**1.4 Configure connection pooling (Important!)**
```
1. Still in Database settings
2. Find "Connection pooling" section
3. Enable "Use connection pooling"
4. Mode: Transaction
5. Copy pooler connection string:
   postgresql://postgres:[PASSWORD]@db.xxxxx.supabase.co:6543/postgres

Note: Use port 6543 for pooler (not 5432 direct)
```

### Step 2: Create Upstash Redis

**2.1 Sign up for Upstash**
```
1. Go to https://upstash.com
2. Sign in with GitHub or email
3. Confirm email address
```

**2.2 Create Redis database**
```
1. Click "Create Database"
2. Name: shopogoda-cache
3. Type: Regional
4. Region: Choose same as Supabase (or closest)
5. TLS: Enabled (recommended)
6. Click "Create"
```

**2.3 Get connection details**
```
1. Open your Redis database
2. Go to "Details" tab
3. Copy these values:

   REST API (recommended for serverless):
   - UPSTASH_REDIS_REST_URL: https://xxxxx.upstash.io
   - UPSTASH_REDIS_REST_TOKEN: AxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxQ==

   Redis Protocol (for Railway):
   - Endpoint: redis-xxxxx.upstash.io
   - Port: 6379 or 6380 (TLS)
   - Password: AxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxQ==
```

### Step 3: Deploy Bot to Railway

**3.1 Create Railway project (same as Method 1)**
```
1. Go to https://railway.app
2. Login with GitHub
3. New Project → Deploy from GitHub repo
4. Choose shopogoda repository
```

**3.2 Configure environment variables**

Click on bot service → Variables tab:

**Required Variables:**
```bash
# Telegram & Weather API
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
OPENWEATHER_API_KEY=your_api_key_from_openweathermap

# Bot Mode
BOT_WEBHOOK_MODE=true
BOT_WEBHOOK_URL=${{RAILWAY_PUBLIC_DOMAIN}}
BOT_WEBHOOK_PORT=8080

# Supabase PostgreSQL (use connection pooler!)
DATABASE_URL=postgresql://postgres.[PROJECT_ID]:[PASSWORD]@aws-X-REGION.pooler.supabase.com:6543/postgres?sslmode=require
DB_HOST=aws-X-REGION.pooler.supabase.com
DB_PORT=6543
DB_NAME=postgres
DB_USER=postgres.[PROJECT_ID]
DB_PASSWORD=your_supabase_db_password
DB_SSL_MODE=require

# Upstash Redis - TCP Protocol with TLS (ShoPogoda uses standard Redis client)
REDIS_HOST=special-name-12345.upstash.io
REDIS_PORT=6379
REDIS_PASSWORD=AxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxQ==
REDIS_DB=0

# Optional
LOG_LEVEL=info
LOG_FORMAT=json
BOT_DEBUG=false
```

**Important notes:**
- ✅ Use Supabase **connection pooler** (port 6543) not direct connection (5432)
- ✅ Set `DB_SSL_MODE=require` for Supabase
- ✅ **ShoPogoda automatically enables TLS for Redis** when connecting to cloud hosts (non-localhost)
- ✅ Use Upstash **TCP endpoint** (not REST API) - standard Redis protocol on port 6379
- ✅ Store passwords securely in Railway variables (never commit to git)
- ⚠️ **Database migrations are disabled** for Supabase pooler compatibility (tables auto-created on first deployment)

**3.3 Generate domain and deploy**
```
1. Bot service → Settings → Networking → Generate Domain
2. Copy generated domain
3. Update BOT_WEBHOOK_URL if needed
4. Railway auto-deploys on variable changes
5. Wait 3-5 minutes for deployment
```

### Step 4: Verify Deployment

**4.1 Check service health**
```bash
# Health check
curl https://your-app.up.railway.app/health

# Expected: {"status":"healthy","timestamp":"..."}
```

**4.2 Check database connection**
```bash
# Via Railway logs
railway logs --service shopogoda

# Look for:
# "Database connected successfully"
# "Redis connected successfully"
# "Bot started successfully"
```

**4.3 Verify webhook**
```bash
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"

# Should show:
# "url": "https://your-app.up.railway.app/webhook"
# "pending_update_count": 0
```

**4.4 Test bot commands**
```
Telegram → Your bot
/start      → Welcome message
/weather    → Current weather (tests API + caching)
/forecast   → 5-day forecast
/setlocation → Set location (tests database write)
```

### Step 5: Monitor Free Tier Usage

**Railway Usage:**
```bash
railway usage

# Monitor:
# - Hours used / 500 free hours
# - Should stay at $0 if under 500 hours/month
```

**Supabase Usage:**
```
1. Supabase Dashboard → Project → Settings → Usage
2. Monitor:
   - Database size: Max 500 MB
   - Bandwidth: Max 2 GB/month
   - Active connections: Max 100
```

**Upstash Usage:**
```
1. Upstash Dashboard → Database → Metrics
2. Monitor:
   - Daily requests: Max 10,000 commands/day
   - Storage: Max 256 MB
   - Bandwidth: Free tier covers most usage
```

### Troubleshooting Hybrid Setup

**Issue 1: YAML parsing errors on deployment**
```bash
# Symptom: "error reading config file: While parsing config: yaml: control characters are not allowed"
# Cause: Railway caching old config files or malformed YAML

# Fix: ShoPogoda now disables YAML config loading in production
# All configuration comes from environment variables only
# No action needed - fixed in latest version
```

**Issue 2: Database connection timeout**
```bash
# Symptom: "connection timeout" in logs
# Cause: Using direct connection instead of pooler

# Fix: Ensure using port 6543 (pooler) not 5432
DB_HOST=aws-X-REGION.pooler.supabase.com
DB_PORT=6543  # Pooler port!
DB_SSL_MODE=require
```

**Issue 3: Prepared statement errors**
```bash
# Symptom: "ERROR: prepared statement already exists (SQLSTATE 42P05)"
# Cause: Supabase transaction pooler doesn't support prepared statements

# Fix: ShoPogoda automatically disables prepared statements for Supabase
# Uses PreferSimpleProtocol=true when connecting
# No action needed - fixed in latest version
```

**Issue 4: Database migration "insufficient arguments" error**
```bash
# Symptom: "Failed to create bot: failed to connect to database: failed to run migrations: insufficient arguments"
# Cause: GORM AutoMigrate schema verification incompatible with Supabase pooler

# Fix: Database migrations are disabled for Railway deployments
# Tables are created automatically on first successful deployment
# If tables don't exist, manually create them once:
psql "postgresql://postgres.[PROJECT]:[PASSWORD]@aws-X-REGION.pooler.supabase.com:6543/postgres" <<EOF
-- Tables are created automatically by bot on startup
-- This is just a fallback if needed
EOF
```

**Issue 5: Redis connection refused or I/O error**
```bash
# Symptom: "connection refused", "I/O error", or "Server closed the connection"
# Cause: Upstash requires TLS for TCP connections

# Fix: ShoPogoda automatically enables TLS for non-localhost Redis hosts
# Ensure using correct Upstash TCP endpoint:
REDIS_HOST=special-name-12345.upstash.io
REDIS_PORT=6379  # Standard port (TLS auto-enabled)
REDIS_PASSWORD=your_upstash_password

# NOT the REST API:
# ❌ UPSTASH_REDIS_REST_URL (not supported)
# ❌ UPSTASH_REDIS_REST_TOKEN (not supported)
```

**Issue 3: Exceeded Supabase bandwidth**
```bash
# Symptom: Database connection fails after high usage

# Check usage:
# Supabase Dashboard → Usage → Bandwidth

# Solutions:
# 1. Optimize queries (reduce data fetched)
# 2. Increase Redis caching TTL
# 3. Clean old weather data more frequently
# 4. Upgrade Supabase plan ($25/month)
```

**Issue 4: Exceeded Upstash command limit**
```bash
# Symptom: Redis commands fail after 10K/day

# Check usage:
# Upstash Dashboard → Metrics → Daily Requests

# Solutions:
# 1. Increase cache TTL (reduce cache writes)
# 2. Batch Redis operations
# 3. Upgrade Upstash plan ($10/month for 100K/day)
```

### Cost Comparison: Integrated vs Hybrid

| Component | Integrated (Railway) | Hybrid (Railway+Supabase+Upstash) |
|-----------|---------------------|----------------------------------|
| **Bot Hosting** | $0 (500 hrs free) | $0 (500 hrs free) |
| **PostgreSQL** | $5/month | $0 (500MB free) |
| **Redis** | $3/month | $0 (10K cmd/day free) |
| **Total** | **$8/month** | **$0/month** |
| **Setup Complexity** | ⭐ Simple | ⭐⭐⭐ Moderate |
| **Latency** | ⚡ Low (same DC) | ⚡⚡ Higher (external) |
| **Scalability** | 📈 Good | 📈📈 Limited by free tiers |
| **Maintenance** | 🔧 Low | 🔧🔧 Medium (3 platforms) |

**Recommendation:**
- **Start with Integrated** if budget allows ($8/month)
- **Switch to Hybrid** if costs are concern or testing
- **Monitor free tier limits** closely in Hybrid setup

### Migration Between Variants

**From Integrated to Hybrid:**
```bash
# 1. Backup Railway database
railway run --service postgres pg_dump > backup.sql

# 2. Restore to Supabase
psql "postgresql://postgres:[PASSWORD]@db.xxxxx.supabase.co:5432/postgres" < backup.sql

# 3. Update Railway variables to point to Supabase/Upstash
# 4. Delete Railway database services
```

**From Hybrid to Integrated:**
```bash
# 1. Backup Supabase database
pg_dump "postgresql://postgres:[PASSWORD]@db.xxxxx.supabase.co:5432/postgres" > backup.sql

# 2. Add Railway PostgreSQL service
railway add --database postgres

# 3. Restore database
railway run --service postgres psql < backup.sql

# 4. Update variables to use Railway services
# 5. Can keep Supabase/Upstash as backup
```

## Detailed Setup

Create `railway.toml` in project root to customize deployment:

```toml
[build]
# Use Dockerfile from docker/ directory
builder = "DOCKERFILE"
dockerfilePath = "docker/Dockerfile"

[deploy]
# Start command (optional, CMD in Dockerfile is used by default)
# startCommand = "./shopogoda"

# Health check (Railway will ping this endpoint)
healthcheckPath = "/health"
healthcheckTimeout = 30

# Restart policy
restartPolicyType = "ON_FAILURE"
restartPolicyMaxRetries = 10
```

**When to add railway.toml:**
- Railway doesn't auto-detect Dockerfile (non-standard location)
- Custom build args needed
- Health check configuration required
- Restart policy customization

### Project Structure for Railway

ShoPogoda works out-of-the-box with Railway because:

1. ✅ **Dockerfile** located at `docker/Dockerfile` (detected automatically with railway.toml)
2. ✅ **Health endpoint** at `/health` (Railway pings every 30s)
3. ✅ **PORT 8080** exposed (Railway injects PORT env variable, app uses it)
4. ✅ **Multi-stage build** (efficient Docker image, faster deployments)
5. ✅ **Auto-connects services** via reference variables (${{Postgres.*}})

### Railway Variables Reference

Railway automatically injects these variables when you add services:

**PostgreSQL:**
```bash
${{Postgres.DATABASE_URL}}       # Full connection string
${{Postgres.PGHOST}}             # Database host
${{Postgres.PGPORT}}             # Database port (5432)
${{Postgres.PGDATABASE}}         # Database name
${{Postgres.PGUSER}}             # Database user
${{Postgres.PGPASSWORD}}         # Database password
```

**Redis:**
```bash
${{Redis.REDIS_URL}}             # Full connection string
${{Redis.REDIS_HOST}}            # Redis host
${{Redis.REDIS_PORT}}            # Redis port (6379)
${{Redis.REDIS_PASSWORD}}        # Redis password
```

**ShoPogoda automatically uses these variables** - no code changes needed!

### Custom Domain (Optional)

**Free Railway subdomain:**
- Format: `your-app.up.railway.app`
- Automatic HTTPS

**Custom domain:**
```bash
# Via CLI
railway domain add yourdomain.com

# Via dashboard
# Settings → Networking → Custom Domain → Add yourdomain.com
# Add CNAME record: yourdomain.com → your-app.up.railway.app
```

## Configuration

### Environment Variables

**Required:**
```bash
TELEGRAM_BOT_TOKEN       # From @BotFather
OPENWEATHER_API_KEY      # From openweathermap.org
```

**Auto-configured by Railway:**
```bash
DATABASE_URL             # PostgreSQL connection string
DB_HOST, DB_PORT, etc.   # Individual DB params
REDIS_URL                # Redis connection string
REDIS_HOST, etc.         # Individual Redis params
PORT                     # App port (Railway sets this)
```

**Optional:**
```bash
LOG_LEVEL                # info, debug, error (default: info)
BOT_DEBUG                # true, false (default: false)
BOT_WEBHOOK_URL          # Auto-set to Railway domain
SLACK_WEBHOOK_URL        # For enterprise notifications
DEMO_MODE                # true, false (default: false)
```

### Railway Service Configuration

**CPU & Memory:**
- Default: 512MB RAM, 1 vCPU shared
- Sufficient for small bots (<10K users)
- Can increase if needed (costs more)

**Scaling:**
- Default: 1 instance
- Auto-restarts on crash
- Horizontal scaling available (paid)

**Health Checks:**
- Railway pings `/health` endpoint
- Auto-restarts if unhealthy
- 30-second timeout

### Database Configuration

**PostgreSQL:**
- Version: 15 (latest)
- Storage: 1 GB (expandable)
- Max connections: 22
- Automatic backups: Daily

**Redis:**
- Version: 7
- Memory: 256 MB (expandable)
- Max connections: 10
- No persistence (cache only)

## Monitoring

### Railway Dashboard

**Available Metrics:**
- CPU usage (%)
- Memory usage (MB)
- Network bandwidth (GB)
- Request count
- Response times
- Error rates

**Access:**
1. Go to [railway.app/dashboard](https://railway.app/dashboard)
2. Select your project
3. Click "Metrics" tab

### Logs

**View logs:**
```bash
# CLI
railway logs

# Follow logs in real-time
railway logs --follow

# Filter by service
railway logs --service shopogoda
```

**Web Dashboard:**
- Project → Service → "Logs" tab
- Real-time log streaming
- Search and filter
- Download logs

### Alerts

**Set up usage alerts:**
1. Project Settings → Usage
2. Set alert threshold (e.g., $4)
3. Add email for notifications

### Health Monitoring

**Built-in health check:**
- Railway automatically pings `/health`
- Restarts service if unhealthy
- View health status in dashboard

**Manual health check:**
```bash
curl https://your-app.up.railway.app/health

# Expected response:
# {"status":"healthy","timestamp":"2025-01-03T..."}
```

## Deployment Workflows

### Auto-Deploy from GitHub

**Setup:**
1. Connect GitHub repository
2. Select branch (main, develop)
3. Enable auto-deploy

**How it works:**
- Push to GitHub → Railway detects change
- Automatically builds Docker image
- Deploys new version
- Zero-downtime rollout

**Rollback:**
```bash
# Via CLI
railway rollback

# Via dashboard
# Deployments → Select previous deployment → Redeploy
```

### Manual Deployment

**From local code:**
```bash
railway up
```

**From Docker image:**
```bash
# Build locally
docker build -t shopogoda:latest .

# Push to Railway
railway up --image shopogoda:latest
```

### Environment-Based Deployments

**Development:**
```bash
railway environment create development
railway environment use development
railway up
```

**Production:**
```bash
railway environment use production
railway up
```

## Troubleshooting

### Common Issues

**1. Bot not responding**

```bash
# Check logs
railway logs --follow

# Common causes:
# - Webhook not set
# - Environment variables missing
# - Database connection failed

# Fix: Verify webhook
curl "https://api.telegram.org/bot<TOKEN>/getWebhookInfo"

# Should show your Railway URL
```

**2. Database connection errors**

```bash
# Verify database variables
railway variables

# Check database status
railway status

# Restart database
railway restart --service postgres
```

**3. Redis connection failed**

```bash
# Check Redis status
railway status

# Verify Redis variables
railway variables | grep REDIS

# Restart Redis
railway restart --service redis
```

**4. Exceeded free tier**

```bash
# Check usage
railway usage

# View current bill
railway billing

# Optimize:
# - Reduce instance size
# - Implement better caching
# - Scale to zero when idle
```

**5. Build failures**

```bash
# View build logs
railway logs --build

# Common causes:
# - Dockerfile errors
# - Missing dependencies
# - Build timeout

# Fix: Check Dockerfile and build process
```

### Debug Mode

**Enable debug logging:**
```bash
railway variables set LOG_LEVEL=debug
railway variables set BOT_DEBUG=true

# Restart to apply
railway restart
```

**View detailed logs:**
```bash
railway logs --follow --service shopogoda
```

### Performance Issues

**Slow response times:**
```bash
# Check metrics
railway metrics

# Solutions:
# - Increase instance size (512MB → 1GB)
# - Add Redis caching
# - Optimize database queries
```

**High memory usage:**
```bash
# View memory usage
railway metrics --memory

# Solutions:
# - Restart service (clears memory)
# - Fix memory leaks in code
# - Upgrade to larger instance
```

## Cost Optimization Tips

### Stay Within Free Tier

**1. Scale to Zero**
- Railway scales automatically
- No cost when bot is idle
- Quick startup on new requests

**2. Efficient Caching**
- Use Redis for weather data
- Reduce OpenWeatherMap API calls
- Cache geocoding results (24 hours)

**3. Database Optimization**
- Clean old weather data (30 days)
- Archive old alerts (90 days)
- Regular VACUUM for PostgreSQL

**4. Optimize Docker Image**
- Use multi-stage builds (already done)
- Minimize image size
- Reduce build time

### Monitor Usage

```bash
# Check monthly usage
railway usage

# View detailed breakdown
railway billing --detailed

# Set alerts at $4 (before hitting $5 limit)
railway alerts set --threshold 4
```

### Cost-Saving Alternatives

**If exceeding free tier or want to minimize costs:**

#### Option 1: Railway + Supabase + Upstash (Hybrid - $0/month)

**Best for:** Maximum cost savings while keeping Railway deployment

**Setup:**
1. Use Railway only for bot (500 free hours = ~20 days always-on)
2. Use Supabase for PostgreSQL (500MB free + 2GB bandwidth)
3. Use Upstash for Redis (10K commands/day free)

**Pros:**
- ✅ **Completely free** for small bots
- ✅ Railway deployment simplicity
- ✅ Professional managed databases
- ✅ No cold starts for database access

**Cons:**
- ❌ Need to manage 3 platforms
- ❌ External database latency
- ❌ Free tier limits (storage, bandwidth)

**See:** [Deployment Variant 2: Railway + External Services](#deployment-variant-2-railway--supabase--upstash) below for detailed setup.

#### Option 2: Fly.io + Supabase + Upstash ($0/month)

**Best for:** Completely free deployment with more control

**Setup:**
- Fly.io free tier: 3 VMs (256MB each)
- Use external databases (same as Option 1)
- Total cost: $0

**See:** [DEPLOYMENT_FLYIO.md](DEPLOYMENT_FLYIO.md) for details.

#### Option 3: Self-Hosted VPS ($4-5/month)

**Best for:** Full control, no platform limits

**Providers:**
- Hetzner Cloud: $4.15/month (2 vCPU, 2GB RAM)
- Contabo: $4.50/month (4 vCPU, 8GB RAM)
- DigitalOcean: $6/month (1 vCPU, 1GB RAM)

**Pros:**
- ✅ Full control over infrastructure
- ✅ No platform limits
- ✅ Can run multiple projects

**Cons:**
- ❌ Manual setup and maintenance
- ❌ Security responsibility
- ❌ No managed backups

## Migration & Backup

### Backup Database

**Manual backup:**
```bash
# Export via CLI
railway run --service postgres pg_dump > backup.sql

# Via Railway dashboard
# PostgreSQL service → Backups → Create backup
```

**Automated backups:**
- Railway automatically backs up daily
- Retention: 7 days
- Restore from dashboard

### Migrate to Another Platform

**Export data:**
```bash
# Database dump
railway run pg_dump $DATABASE_URL > shopogoda_backup.sql

# Environment variables
railway variables > variables.env
```

**Import to new platform:**
```bash
# Restore database
psql $NEW_DATABASE_URL < shopogoda_backup.sql

# Set environment variables
# (platform-specific)
```

## Next Steps

1. ✅ Deploy bot to Railway
2. ✅ Set webhook
3. ✅ Test bot functionality
4. 🔄 Monitor usage and logs
5. 🔄 Set up usage alerts
6. 🔄 Configure custom domain (optional)
7. 🔄 Enable GitHub auto-deploy

## Resources

- [Railway Documentation](https://docs.railway.app)
- [Railway Discord](https://discord.gg/railway)
- [Railway Status](https://status.railway.app)
- [Railway Blog](https://blog.railway.app)

## Support

**Railway Support:**
- Discord: [discord.gg/railway](https://discord.gg/railway)
- Email: team@railway.app
- Docs: [docs.railway.app](https://docs.railway.app)

**ShoPogoda Issues:**
- GitHub: [github.com/valpere/shopogoda/issues](https://github.com/valpere/shopogoda/issues)
- Email: valentyn.solomko@gmail.com

---

**Last Updated**: 2025-01-07
**Railway Version**: v2 (2025 pricing model)
**Maintained by**: [@valpere](https://github.com/valpere)
**ShoPogoda Version**: Compatible with v1.0+
