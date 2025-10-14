# Deploying ShoPogoda to Vercel

This guide provides step-by-step instructions for deploying the ShoPogoda Telegram bot to Vercel's serverless platform with external PostgreSQL (Supabase) and Redis (Upstash).

## Overview

**Architecture**: Vercel Serverless Functions + Supabase (PostgreSQL) + Upstash (Redis)

**Total Cost**: **$0/month** (within free tier limits)

### Why Vercel?

- ✅ **Truly Free**: 100GB bandwidth, 100GB-Hours compute, unlimited projects
- ✅ **Serverless**: Pay only for actual usage, scales to zero automatically
- ✅ **Fast Deployment**: Deploy from GitHub in seconds
- ✅ **Global Edge Network**: Low latency worldwide
- ✅ **Built-in CI/CD**: Automatic deployments on git push
- ✅ **Perfect for Webhooks**: Event-driven architecture ideal for Telegram bots

### Free Tier Limits

| Resource | Hobby (Free) | Pro |
|----------|--------------|-----|
| **Bandwidth** | 100 GB/month | 1 TB/month |
| **Function Invocations** | 100 GB-Hours | 1000 GB-Hours |
| **Build Minutes** | 6,000/month | Unlimited |
| **Projects** | Unlimited | Unlimited |
| **Deployments** | Unlimited | Unlimited |
| **Team Members** | 1 | Unlimited |
| **Log Retention** | 1 hour | 3 days |

**Overage Costs**:

- Bandwidth: $20 per 100 GB
- Compute: $20 per 100 GB-Hours

**Note**: For a typical weather bot with moderate usage (100-500 users), free tier is more than sufficient.

---

## Prerequisites

### Required Accounts

1. **Vercel Account** (free forever)
   - Sign up at: <https://vercel.com/signup>
   - GitHub OAuth recommended for easy deployment

2. **Supabase Account** (PostgreSQL - free forever)
   - Sign up at: <https://supabase.com>
   - Free tier: 500MB database, 50K MAU, unlimited API requests

3. **Upstash Account** (Redis - free forever)
   - Sign up at: <https://upstash.com>
   - Free tier: 10,000 commands/day, 256MB storage

4. **Telegram Bot Token**
   - Create bot via @BotFather on Telegram
   - Save your token (format: `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`)

5. **OpenWeatherMap API Key**
   - Sign up at: <https://openweathermap.org>
   - Free tier: 60 calls/minute, 1M calls/month

### Required Tools

- **Git** installed locally
- **Vercel CLI** (optional, for local development):

  ```bash
  npm install -g vercel
  ```

---

## Step 1: Set Up Supabase PostgreSQL

### 1.1 Create Supabase Project

1. Go to <https://supabase.com/dashboard>
2. Click **"New project"**
3. Configure project:
   - **Name**: `shopogoda-db`
   - **Database Password**: Generate a strong password (save it!)
   - **Region**: Choose closest to your users (e.g., `us-east-1`, `eu-central-1`)
   - **Pricing Plan**: Free

4. Wait 2-3 minutes for provisioning

### 1.2 Get Connection Details

1. Go to **Project Settings** → **Database**
2. Find **Connection Pooling** section (important for serverless!)
3. Copy these values:

```bash
# Connection Pooler (use this for serverless - port 6543)
Host: db.abcdefghijklmnop.supabase.co
Port: 6543  # Note: NOT 5432
Database: postgres
User: postgres
Password: [your password]
```

4. Construct connection string:

```bash
postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:6543/postgres
```

**Why Connection Pooler?**

- Serverless functions create many short-lived connections
- Connection pooler (port 6543) handles this efficiently
- Direct connections (port 5432) will cause "too many connections" errors

### 1.3 Configure Supabase

1. Go to **SQL Editor**
2. Enable required extensions:

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable PostGIS for location data (optional but recommended)
CREATE EXTENSION IF NOT EXISTS postgis;
```

3. Click **Run** to execute

---

## Step 2: Set Up Upstash Redis

### 2.1 Create Redis Database

1. Go to <https://console.upstash.com/redis>
2. Click **"Create database"**
3. Configure database:
   - **Name**: `shopogoda-redis`
   - **Type**: Regional (faster) or Global (multi-region)
   - **Region**: Choose closest to your Vercel deployment region
   - **TLS**: Enabled (recommended)

4. Click **"Create"**

### 2.2 Get Connection Details

1. On database details page, find **REST API** section (recommended for serverless)
2. Copy these values:

```bash
# REST API (recommended for Vercel)
UPSTASH_REDIS_REST_URL=https://us1-adapted-bat-12345.upstash.io
UPSTASH_REDIS_REST_TOKEN=AbCdEf123456...

# Or use Redis Protocol
REDIS_URL=rediss://default:[PASSWORD]@us1-adapted-bat-12345.upstash.io:6379
```

**Why REST API?**

- Serverless-friendly (connection pooling not needed)
- Lower latency for short-lived functions
- Better for Vercel's execution model

### 2.3 Test Connection (Optional)

```bash
curl $UPSTASH_REDIS_REST_URL/ping \
  -H "Authorization: Bearer $UPSTASH_REDIS_REST_TOKEN"

# Expected response: {"result":"PONG"}
```

---

## Step 3: Prepare Your Repository

### 3.1 Restructure for Vercel

Vercel expects serverless functions in the `api/` directory. We need to adapt our Go application:

**Option A: Webhook Handler (Recommended)**

Create `api/webhook.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/database"
	"github.com/valpere/shopogoda/internal/handlers/commands"
	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
)

var (
	botInstance *gotgbot.Bot
	dispatcher  *ext.Dispatcher
	servicesInstance *services.Services
	initialized bool
)

// Handler is the Vercel serverless function entry point
func Handler(w http.ResponseWriter, r *http.Request) {
	// Initialize on first request (cold start)
	if !initialized {
		if err := initialize(); err != nil {
			log.Error().Err(err).Msg("Failed to initialize bot")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		initialized = true
	}

	// Only accept POST requests to /webhook
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse Telegram update
	var update gotgbot.Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Error().Err(err).Msg("Failed to parse update")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Process update
	if err := dispatcher.ProcessUpdate(botInstance, &update, nil); err != nil {
		log.Error().Err(err).Msg("Failed to process update")
		// Don't return error to Telegram - they'll retry
	}

	// Always return 200 OK to Telegram
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func initialize() error {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// Load configuration from environment
	cfg := &config.Config{
		Bot: config.BotConfig{
			Token:       os.Getenv("TELEGRAM_BOT_TOKEN"),
			Debug:       os.Getenv("BOT_DEBUG") == "true",
			WebhookMode: true,
			WebhookURL:  os.Getenv("WEBHOOK_URL"),
		},
		Database: config.DatabaseConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  "require",
		},
		Redis: config.RedisConfig{
			Host:     os.Getenv("REDIS_HOST"),
			Port:     os.Getenv("REDIS_PORT"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       0,
		},
		Weather: config.WeatherConfig{
			APIKey:      os.Getenv("OPENWEATHER_API_KEY"),
			BaseURL:     "https://api.openweathermap.org/data/2.5",
			CacheTTL:    10 * time.Minute,
			GeocodeTTL:  24 * time.Hour,
			ForecastTTL: 1 * time.Hour,
		},
	}

	// Validate required fields
	if cfg.Bot.Token == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.Weather.APIKey == "" {
		return fmt.Errorf("OPENWEATHER_API_KEY is required")
	}
	if cfg.Bot.WebhookURL == "" {
		return fmt.Errorf("WEBHOOK_URL is required")
	}

	// Initialize database
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := models.Migrate(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize Redis
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to connect to Redis, continuing without cache")
		redisClient = nil
	}

	// Initialize services
	servicesInstance = services.New(db, redisClient, cfg, &log.Logger)

	// Create bot instance
	botInstance, err = gotgbot.NewBot(cfg.Bot.Token, &gotgbot.BotOpts{})
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	// Create dispatcher
	dispatcher = ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Error().Err(err).Msg("Update processing error")
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	// Register command handlers
	registerHandlers(dispatcher, servicesInstance)

	log.Info().Msg("Bot initialized successfully")
	return nil
}

func registerHandlers(dispatcher *ext.Dispatcher, services *services.Services) {
	// Command handlers
	dispatcher.AddHandler(commands.NewStartHandler(services))
	dispatcher.AddHandler(commands.NewWeatherHandler(services))
	dispatcher.AddHandler(commands.NewForecastHandler(services))
	dispatcher.AddHandler(commands.NewAirHandler(services))
	dispatcher.AddHandler(commands.NewSetLocationHandler(services))
	dispatcher.AddHandler(commands.NewSubscribeHandler(services))
	dispatcher.AddHandler(commands.NewAddAlertHandler(services))
	dispatcher.AddHandler(commands.NewSettingsHandler(services))
	dispatcher.AddHandler(commands.NewStatsHandler(services))
	dispatcher.AddHandler(commands.NewBroadcastHandler(services))
	dispatcher.AddHandler(commands.NewUsersHandler(services))

	// Callback query handlers
	dispatcher.AddHandler(commands.NewCallbackQueryHandler(services))
}
```

Create `vercel.json` (Vercel configuration):

```json
{
  "version": 2,
  "builds": [
    {
      "src": "api/webhook.go",
      "use": "@vercel/go"
    }
  ],
  "routes": [
    {
      "src": "/webhook",
      "dest": "/api/webhook"
    },
    {
      "src": "/health",
      "dest": "/api/health"
    }
  ],
  "env": {
    "GO111MODULE": "on",
    "CGO_ENABLED": "0"
  }
}
```

Create `api/health.go` (health check endpoint):

```go
package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
```

### 3.2 Update go.mod

Ensure your `go.mod` specifies Go 1.21 or later:

```go
module github.com/valpere/shopogoda

go 1.21

require (
	github.com/PaulSonOfLars/gotgbot/v2 v2.0.0-rc.25
	github.com/gin-gonic/gin v1.9.1
	github.com/go-redis/redis/v9 v9.0.5
	github.com/rs/zerolog v1.29.1
	github.com/spf13/viper v1.15.0
	gorm.io/driver/postgres v1.5.2
	gorm.io/gorm v1.25.1
)
```

### 3.3 Commit Changes

```bash
git add api/ vercel.json
git commit -m "feat: Add Vercel serverless function support"
git push origin main
```

---

## Step 4: Deploy to Vercel

### 4.1 Deploy from GitHub (Recommended)

1. Go to <https://vercel.com/new>
2. Click **"Import Git Repository"**
3. Select your `shopogoda` repository
4. Configure project:
   - **Framework Preset**: Other
   - **Root Directory**: `./` (leave default)
   - **Build Command**: (leave empty - Vercel auto-detects Go)
   - **Output Directory**: (leave empty)

5. Click **"Deploy"**

### 4.2 Configure Environment Variables

**Before first deployment**, add environment variables:

1. In Vercel dashboard, go to **Settings** → **Environment Variables**
2. Add the following variables:

| Name | Value | Environment |
|------|-------|-------------|
| `TELEGRAM_BOT_TOKEN` | `123456:ABC-DEF...` | Production, Preview, Development |
| `OPENWEATHER_API_KEY` | `your-openweather-key` | Production, Preview, Development |
| `WEBHOOK_URL` | `https://your-app.vercel.app/webhook` | Production |
| `DB_HOST` | `db.abcdefgh.supabase.co` | Production, Preview, Development |
| `DB_PORT` | `6543` | Production, Preview, Development |
| `DB_USER` | `postgres` | Production, Preview, Development |
| `DB_PASSWORD` | `your-supabase-password` | Production, Preview, Development |
| `DB_NAME` | `postgres` | Production, Preview, Development |
| `REDIS_HOST` | `us1-adapted-bat-12345.upstash.io` | Production, Preview, Development |
| `REDIS_PORT` | `6379` | Production, Preview, Development |
| `REDIS_PASSWORD` | `your-upstash-password` | Production, Preview, Development |
| `LOG_LEVEL` | `info` | Production, Preview, Development |
| `BOT_DEBUG` | `false` | Production |

**Tip**: Use **Sensitive** checkbox for secrets (TELEGRAM_BOT_TOKEN, API keys, passwords).

3. Click **"Save"**
4. Redeploy: **Deployments** → **⋮** → **Redeploy**

### 4.3 Get Deployment URL

After deployment completes:

1. Vercel provides a URL: `https://shopogoda-xyz123.vercel.app`
2. Copy this URL
3. Update `WEBHOOK_URL` environment variable with this URL + `/webhook`:

   ```
   https://shopogoda-xyz123.vercel.app/webhook
   ```

4. Redeploy again

### 4.4 Alternative: Deploy with Vercel CLI

```bash
# Login to Vercel
vercel login

# Deploy
vercel

# Follow prompts:
# Set up and deploy? [Y/n] y
# Which scope? [your-account]
# Link to existing project? [n] n
# Project name? shopogoda
# Directory? ./
# Override settings? [n] n

# Set environment variables
vercel env add TELEGRAM_BOT_TOKEN
vercel env add OPENWEATHER_API_KEY
# ... (add all variables)

# Deploy to production
vercel --prod
```

---

## Step 5: Configure Telegram Webhook

### 5.1 Set Webhook URL

Use Telegram's setWebhook API to point your bot to Vercel:

```bash
# Replace with your values
BOT_TOKEN="123456:ABC-DEF..."
WEBHOOK_URL="https://shopogoda-xyz123.vercel.app/webhook"

# Set webhook
curl -X POST "https://api.telegram.org/bot${BOT_TOKEN}/setWebhook" \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"${WEBHOOK_URL}\"}"

# Expected response:
# {"ok":true,"result":true,"description":"Webhook was set"}
```

### 5.2 Verify Webhook

```bash
# Check webhook info
curl "https://api.telegram.org/bot${BOT_TOKEN}/getWebhookInfo"

# Expected response:
# {
#   "ok": true,
#   "result": {
#     "url": "https://shopogoda-xyz123.vercel.app/webhook",
#     "has_custom_certificate": false,
#     "pending_update_count": 0,
#     "max_connections": 40
#   }
# }
```

### 5.3 Test Health Endpoint

```bash
curl https://shopogoda-xyz123.vercel.app/health

# Expected response:
# {
#   "status": "ok",
#   "timestamp": "2025-10-04T12:34:56Z",
#   "version": "1.0.0"
# }
```

---

## Step 6: Test Your Bot

### 6.1 Send Test Message

1. Open Telegram
2. Search for your bot: `@YourBotName`
3. Send `/start`

**Expected response**:

```
🌤 Welcome to ShoPogoda!

I'm your personal weather assistant...
```

### 6.2 Test Weather Command

```
/weather New York
```

**Expected response**: Current weather for New York with temperature, conditions, etc.

### 6.3 Check Vercel Logs

1. Go to Vercel dashboard → **Deployments**
2. Click on your deployment
3. Go to **Functions** tab
4. Click on `/api/webhook` function
5. View real-time logs

**Example logs**:

```
2025-10-04T12:34:56.000Z INFO Bot initialized successfully
2025-10-04T12:35:12.000Z INFO Processing update from user 123456
2025-10-04T12:35:13.000Z INFO Weather command executed successfully
```

---

## Step 7: Configure Custom Domain (Optional)

### 7.1 Add Domain to Vercel

1. Go to **Settings** → **Domains**
2. Click **"Add"**
3. Enter your domain: `bot.yourdomain.com`
4. Follow DNS configuration instructions

### 7.2 Update Webhook URL

After domain is verified:

```bash
BOT_TOKEN="123456:ABC-DEF..."
WEBHOOK_URL="https://bot.yourdomain.com/webhook"

curl -X POST "https://api.telegram.org/bot${BOT_TOKEN}/setWebhook" \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"${WEBHOOK_URL}\"}"
```

### 7.3 Update Environment Variable

1. Vercel dashboard → **Settings** → **Environment Variables**
2. Update `WEBHOOK_URL` to your custom domain
3. Redeploy

---

## Monitoring and Maintenance

### 7.1 View Logs

**Real-time logs**:

1. Vercel dashboard → **Deployments** → [your deployment]
2. Click **"View Function Logs"**
3. Filter by function: `/api/webhook`

**Note**: Free tier only retains logs for 1 hour.

**Alternative: Use external logging** (recommended for production):

```go
// Send logs to external service (e.g., Logtail, Datadog)
import "github.com/logtail/logtail-go"

logger := logtail.New(os.Getenv("LOGTAIL_TOKEN"))
```

### 7.2 Monitor Usage

1. Vercel dashboard → **Analytics**
2. Check:
   - **Bandwidth**: Should stay under 100 GB/month
   - **Function Invocations**: Should stay under 100 GB-Hours
   - **Build Minutes**: Should stay under 6,000/month

**Typical weather bot usage** (500 active users):

- Bandwidth: ~5-10 GB/month
- Function Invocations: ~20-30 GB-Hours/month
- Well within free tier! ✅

### 7.3 Performance Optimization

**Reduce Cold Starts**:

1. Use **Edge Middleware** for faster responses:

Create `middleware.ts`:

```typescript
import { NextRequest, NextResponse } from 'next/server'

export function middleware(request: NextRequest) {
  // Fast edge response for health checks
  if (request.nextUrl.pathname === '/health') {
    return NextResponse.json({ status: 'ok' })
  }

  return NextResponse.next()
}
```

2. **Keep Functions Warm** (optional, costs extra):

```bash
# Use cron job to ping every 5 minutes (prevents cold starts)
# Add to vercel.json:
{
  "crons": [
    {
      "path": "/api/keepalive",
      "schedule": "*/5 * * * *"
    }
  ]
}
```

3. **Optimize Database Connections**:
   - Use connection pooler (port 6543) - already configured ✅
   - Reuse connections across invocations (singleton pattern)
   - Set aggressive connection timeouts

### 7.4 Supabase Monitoring

1. Go to Supabase dashboard → **Database** → **Database Health**
2. Monitor:
   - **Active connections**: Should stay under 100
   - **Database size**: Free tier limit is 500 MB
   - **API requests**: Unlimited on free tier

**Optimize database**:

```sql
-- Check table sizes
SELECT
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Add indexes for common queries
CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
CREATE INDEX IF NOT EXISTS idx_weather_data_user_created ON weather_data(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_alerts_user_triggered ON environmental_alerts(user_id, triggered_at);
```

### 7.5 Upstash Monitoring

1. Go to Upstash dashboard → [your database]
2. Monitor:
   - **Daily Requests**: Should stay under 10,000
   - **Storage**: Free tier limit is 256 MB
   - **Latency**: Should be <50ms

**Optimize Redis usage**:

- Use appropriate TTLs (10 min for weather, 1 hour for forecasts)
- Avoid storing large objects (>1MB)
- Use compression for large values

---

## Troubleshooting

### Issue: "Too Many Connections" Error

**Cause**: Using direct PostgreSQL connection (port 5432) instead of connection pooler.

**Fix**:

1. Use port **6543** (connection pooler), not 5432
2. Update `DB_PORT` environment variable
3. Redeploy

### Issue: Webhook Not Receiving Updates

**Check webhook status**:

```bash
curl "https://api.telegram.org/bot${BOT_TOKEN}/getWebhookInfo"
```

**Common causes**:

1. **Wrong URL**: Ensure `WEBHOOK_URL` matches your Vercel deployment
2. **SSL certificate**: Vercel provides auto-SSL, should work by default
3. **Firewall**: Telegram servers must be able to reach your Vercel function

**Fix**:

```bash
# Delete old webhook
curl -X POST "https://api.telegram.org/bot${BOT_TOKEN}/deleteWebhook"

# Set new webhook
curl -X POST "https://api.telegram.org/bot${BOT_TOKEN}/setWebhook" \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"https://your-app.vercel.app/webhook\"}"
```

### Issue: Function Timeout (10s limit)

**Cause**: Long-running operations exceed Vercel's 10-second serverless function limit (Hobby plan).

**Fix**:

1. **Optimize database queries**: Add indexes, use connection pooling
2. **Cache aggressively**: Use Redis for weather data (10 min TTL)
3. **Upgrade to Pro**: 60-second timeout ($20/month)
4. **Background jobs**: Use external cron service (e.g., cron-job.org) for scheduled tasks

### Issue: Cold Start Latency

**Symptom**: First request after idle period takes 2-5 seconds.

**Expected behavior**: Vercel serverless functions "sleep" when idle and "wake up" on request.

**Mitigation**:

1. **Accept it**: Most users won't notice 2-5s delay occasionally
2. **Keep warm**: Use cron job to ping every 5 minutes (costs extra bandwidth)
3. **Edge functions**: Migrate to Vercel Edge Functions (faster cold starts)

### Issue: Environment Variables Not Loading

**Check**:

1. Vercel dashboard → **Settings** → **Environment Variables**
2. Ensure variables are set for **Production** environment
3. After changes, **redeploy** (changes don't apply to existing deployments)

**Verify**:

```go
// Add debug logging in api/webhook.go
log.Info().Str("webhook_url", os.Getenv("WEBHOOK_URL")).Msg("Environment check")
```

### Issue: Database Migration Errors

**Symptom**: `relation "users" does not exist`

**Cause**: Migrations not running on first deployment.

**Fix**:

1. Connect to Supabase via SQL Editor
2. Manually run migrations from `internal/models/migrate.go`
3. Or use migration script:

```bash
# Run migrations via Vercel CLI
vercel env pull .env.production
go run cmd/migrate/main.go
```

---

## Cost Optimization

### Stay Within Free Tier

**Monitor usage weekly**:

1. Vercel dashboard → **Usage**
2. Check bandwidth and function invocations
3. Set up alerts when approaching limits

**Typical costs** (500 active users):

- **Bandwidth**: 5-10 GB/month (10% of free tier)
- **Function Invocations**: 20-30 GB-Hours/month (30% of free tier)
- **Supabase**: 50 MB database (10% of free tier)
- **Upstash**: 5,000 commands/day (50% of free tier)

**Result**: Well within all free tiers! ✅

### Reduce Bandwidth

1. **Enable compression** (Vercel enables by default)
2. **Minimize API responses**: Only send necessary data to Telegram
3. **Cache aggressively**: Use Redis to avoid duplicate API calls

### Reduce Function Invocations

1. **Batch operations**: Process multiple users in single function call
2. **Use webhooks** (not polling): Only pay for actual messages
3. **Optimize code**: Reduce execution time = lower GB-Hours

### Database Cost Optimization

**Supabase free tier (500 MB)**:

- Delete old weather data (>30 days): ~10 MB/month saved
- Compress large JSON fields: ~20% size reduction
- Use JSONB instead of TEXT for metadata: ~30% size reduction

**Example cleanup script**:

```sql
-- Delete weather data older than 30 days
DELETE FROM weather_data WHERE created_at < NOW() - INTERVAL '30 days';

-- Delete resolved alerts older than 90 days
DELETE FROM environmental_alerts WHERE resolved_at < NOW() - INTERVAL '90 days';

-- Vacuum to reclaim space
VACUUM FULL;
```

**Schedule cleanup** (run weekly via cron):

```bash
# Add to vercel.json
{
  "crons": [
    {
      "path": "/api/cleanup",
      "schedule": "0 0 * * 0"  # Every Sunday at midnight
    }
  ]
}
```

---

## Comparison: Vercel vs Other Platforms

| Feature | Vercel | Fly.io | Railway | GCP |
|---------|--------|--------|---------|-----|
| **Free Tier** | 100 GB bandwidth | 3 free VMs | $5 credit/month | $300 trial (90 days) |
| **PostgreSQL** | External (Supabase) | External (Supabase) | Included | Cloud SQL ($15/mo) |
| **Redis** | External (Upstash) | External (Upstash) | Included | Memorystore ($25/mo) |
| **Monthly Cost** | **$0** | **$0** | **$0-5** | **$15-20** (after trial) |
| **Deployment** | GitHub auto-deploy | Docker + CLI | GitHub auto-deploy | Cloud Run + SDK |
| **Cold Starts** | 2-5 seconds | None (always-on) | None (always-on) | 1-2 seconds |
| **Scaling** | Auto (serverless) | Manual (VMs) | Auto | Auto (serverless) |
| **Best For** | Low traffic, webhook bots | 24/7 bots, global edge | Hobby projects | Enterprise, high traffic |

**Recommendation**:

- **Vercel**: Best for **low-traffic bots** (< 1000 users) with occasional usage
- **Fly.io**: Best for **24/7 bots** with consistent traffic and no cold starts
- **Railway**: Best for **quick prototypes** and hobby projects
- **GCP**: Best for **enterprise** with budget and advanced features

---

## Next Steps

### 1. Enable Monitoring

**Vercel Analytics** (free):

1. Dashboard → **Analytics**
2. View real-time traffic, function performance

**Uptime Monitoring** (external, free):

- Use UptimeRobot: <https://uptimerobot.com>
- Monitor `/health` endpoint every 5 minutes
- Get alerts on downtime

### 2. Set Up Alerts

**Vercel Integrations**:

1. Dashboard → **Integrations**
2. Add **Slack** or **Discord** for deployment notifications
3. Get alerts on build failures, deployment errors

**Supabase Alerts**:

1. Dashboard → **Project Settings** → **Alerts**
2. Enable alerts for:
   - Database size approaching limit
   - Connection count high

### 3. Backup Strategy

**Database backups** (Supabase automatic):

- Supabase provides automatic daily backups (7-day retention on free tier)
- Manual backups: Dashboard → **Database** → **Backups** → **Create Backup**

**Export data** (optional):

```bash
# Backup via pg_dump
pg_dump "postgresql://postgres:PASSWORD@db.PROJECT.supabase.co:6543/postgres" > backup.sql

# Restore
psql "postgresql://postgres:PASSWORD@db.PROJECT.supabase.co:6543/postgres" < backup.sql
```

### 4. CI/CD Pipeline

Vercel provides automatic CI/CD:

1. Push to `main` → Auto-deploy to production
2. Push to feature branch → Auto-deploy to preview URL
3. Pull request → Auto-deploy to unique preview URL

**Add tests before deploy** (optional):

```json
// vercel.json
{
  "buildCommand": "go test ./... && go build -o /tmp/bot cmd/bot/main.go",
  "ignoreCommand": "git diff HEAD^ HEAD --quiet . ':(exclude).github/'"
}
```

### 5. Performance Testing

**Load test your bot**:

```bash
# Send 100 concurrent requests
for i in {1..100}; do
  curl -X POST "https://your-app.vercel.app/webhook" \
    -H "Content-Type: application/json" \
    -d '{"update_id":1,"message":{"message_id":1,"from":{"id":123,"is_bot":false,"first_name":"Test"},"chat":{"id":123,"type":"private"},"date":1640000000,"text":"/weather"}}' &
done
wait
```

**Expected results**:

- Response time: < 1 second (after cold start)
- Success rate: > 99%
- No errors in logs

---

## Security Best Practices

### 1. Environment Variables

- ✅ **Use Vercel Environment Variables** for all secrets
- ✅ **Never commit** `.env` files to git
- ✅ **Mark sensitive** variables as "Sensitive" in Vercel UI
- ✅ **Rotate secrets** regularly (every 90 days)

### 2. Database Security

**Supabase**:

- ✅ Use **connection pooler** (port 6543) for serverless
- ✅ Enable **SSL mode** (`require` or `verify-full`)
- ✅ Use **Row Level Security (RLS)** for sensitive tables
- ✅ Restrict **IP allowlist** (optional, not needed for most cases)

**Example RLS policy**:

```sql
-- Only allow users to see their own data
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_policy ON users
  FOR ALL
  USING (telegram_id = current_setting('app.current_user_id')::bigint);
```

### 3. Redis Security

**Upstash**:

- ✅ Use **TLS** (enabled by default)
- ✅ Use **strong password** (auto-generated by Upstash)
- ✅ Restrict **access via REST API token** (recommended)

### 4. Telegram Security

- ✅ **Verify webhook requests** (check `X-Telegram-Bot-Api-Secret-Token` header)
- ✅ **Validate user input** before processing
- ✅ **Rate limit** commands (prevent spam)
- ✅ **Use HTTPS** for webhook (required by Telegram)

**Example webhook verification**:

```go
func Handler(w http.ResponseWriter, r *http.Request) {
	// Verify Telegram secret token
	expectedToken := os.Getenv("TELEGRAM_SECRET_TOKEN")
	actualToken := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")

	if expectedToken != "" && actualToken != expectedToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Process update...
}
```

---

## Conclusion

You now have a **completely free** ShoPogoda deployment on Vercel with:

- ✅ **Vercel**: Serverless functions with 100 GB bandwidth/month
- ✅ **Supabase**: 500 MB PostgreSQL with unlimited API requests
- ✅ **Upstash**: 10,000 Redis commands/day with 256 MB storage
- ✅ **Total Cost**: **$0/month** forever (within free tier limits)

**Perfect for**:

- Personal weather bots
- Educational projects
- Low-traffic production bots (< 1000 users)
- Rapid prototyping and testing

**Next steps**:

1. Monitor usage weekly (stay within free tier)
2. Set up uptime monitoring (UptimeRobot)
3. Configure custom domain (optional)
4. Add more features and commands!

For questions or issues, check:

- **Vercel Docs**: <https://vercel.com/docs>
- **Supabase Docs**: <https://supabase.com/docs>
- **Upstash Docs**: <https://docs.upstash.com>
- **ShoPogoda Issues**: <https://github.com/valpere/shopogoda/issues>

Happy deploying! 🚀
