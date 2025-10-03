# Railway Deployment Guide

Complete guide for deploying ShoPogoda to Railway - the easiest and most cost-effective platform for hobby projects.

## Table of Contents

- [Why Railway](#why-railway)
- [Cost Breakdown](#cost-breakdown)
- [Quick Start](#quick-start)
- [Detailed Setup](#detailed-setup)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Why Railway

**Best for:** Hobby projects, quick deployments, minimal configuration

**Advantages:**
- ✅ **$5/month free credit** (enough for small bot + database + Redis)
- ✅ **One-click deployment** from GitHub
- ✅ **PostgreSQL & Redis included** in free tier
- ✅ **Zero configuration** needed
- ✅ **Automatic HTTPS** and domain
- ✅ **Built-in monitoring** and logs
- ✅ **No credit card required** to start

**Perfect for:**
- Personal projects
- Development/testing
- Low-traffic bots (<10K users)
- Learning and experimentation

## Cost Breakdown

### Free Tier Details

**Monthly Allowance:**
- $5 in usage credits (resets monthly)
- Includes: compute, database, Redis, bandwidth

**Typical Bot Usage:**
| Resource | Usage | Cost/Month | Notes |
|----------|-------|------------|-------|
| Bot (512MB RAM) | ~100 hours | $1-2 | Scales to zero when idle |
| PostgreSQL | Always on | $1-2 | db-f1-micro equivalent |
| Redis | Always on | $1 | 256MB instance |
| **Total** | - | **$3-5** | Within free tier! |

**Cost Optimization:**
- Bot scales to zero when no requests → Near $0 compute cost
- Small database footprint → Minimal storage cost
- Redis cache hits reduce external API calls

### After Free Tier ($5 exceeded)

If you exceed $5/month, Railway charges **$0.000231/GB-second** for compute.

**Example for busy bot:**
- Bot: $5/month (512MB, always running)
- PostgreSQL: $3/month
- Redis: $2/month
- **Total: ~$10/month**

**Still cheaper than:**
- GCP Cloud Run + Cloud SQL: $15-20/month
- AWS ECS + RDS: $20-30/month
- DigitalOcean: $12/month minimum

## Quick Start

### Method 1: Web Dashboard (Easiest - 5 Minutes)

**1. Sign Up**
- Go to [railway.app](https://railway.app)
- Click "Login with GitHub"
- Authorize Railway

**2. Create New Project**
- Click "New Project"
- Select "Deploy from GitHub repo"
- Choose `valpere/shopogoda`
- Railway auto-detects Dockerfile

**3. Add Database Services**
- Click "+ New" in your project
- Select "Database" → "PostgreSQL"
- Click "+ New" again
- Select "Database" → "Redis"

**4. Configure Environment Variables**

In your bot service settings, add:

```bash
# Telegram Bot (from @BotFather)
TELEGRAM_BOT_TOKEN=123456789:ABC-DEF1234ghIkl-zyx57W2v1u123ew11

# Weather API (from openweathermap.org)
OPENWEATHER_API_KEY=abcdef1234567890abcdef1234567890

# Database (auto-configured by Railway)
DATABASE_URL=${{Postgres.DATABASE_URL}}
DB_HOST=${{Postgres.PGHOST}}
DB_PORT=${{Postgres.PGPORT}}
DB_NAME=${{Postgres.PGDATABASE}}
DB_USER=${{Postgres.PGUSER}}
DB_PASSWORD=${{Postgres.PGPASSWORD}}

# Redis (auto-configured by Railway)
REDIS_URL=${{Redis.REDIS_URL}}
REDIS_HOST=${{Redis.REDIS_HOST}}
REDIS_PORT=${{Redis.REDIS_PORT}}
REDIS_PASSWORD=${{Redis.REDIS_PASSWORD}}

# Optional
LOG_LEVEL=info
BOT_DEBUG=false
```

**5. Deploy**
- Railway automatically triggers deployment
- Wait 2-3 minutes for build
- Check logs for "Bot started successfully"

**6. Get Bot URL and Set Webhook**

```bash
# Get your Railway domain (shown in deployment)
# Example: shopogoda-production.up.railway.app

# Set webhook using curl
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -d "url=https://shopogoda-production.up.railway.app/webhook"
```

**7. Test Your Bot**
- Open Telegram
- Message your bot: `/start`
- Try: `/weather London`

✅ **Done!** Your bot is live.

### Method 2: Railway CLI (Advanced)

**1. Install Railway CLI**

```bash
# macOS/Linux
curl -fsSL https://railway.app/install.sh | sh

# Or via npm
npm install -g @railway/cli

# Windows (PowerShell)
iwr https://railway.app/install.ps1 | iex
```

**2. Login**

```bash
railway login
```

**3. Initialize Project**

```bash
cd /home/val/wrk/projects/telegram_bot/shopogoda

# Create new Railway project
railway init

# Link to existing project (if created via web)
railway link
```

**4. Add Services**

```bash
# Add PostgreSQL
railway add --database postgres

# Add Redis
railway add --database redis
```

**5. Set Environment Variables**

```bash
# Set bot token
railway variables set TELEGRAM_BOT_TOKEN="your_token_here"

# Set weather API key
railway variables set OPENWEATHER_API_KEY="your_api_key_here"

# Optional settings
railway variables set LOG_LEVEL="info"
railway variables set BOT_DEBUG="false"
```

**6. Deploy**

```bash
railway up
```

**7. View Logs**

```bash
railway logs
```

**8. Get Domain and Set Webhook**

```bash
# Get domain
railway domain

# Set webhook
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -d "url=https://your-domain.railway.app/webhook"
```

## Detailed Setup

### Project Structure for Railway

Railway works out-of-the-box with ShoPogoda because:

1. **Detects Dockerfile** automatically
2. **Exposes PORT** environment variable (app listens on 8080)
3. **Auto-connects services** via reference variables

No additional configuration needed!

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

**If exceeding $5/month:**

1. **Use external free services:**
   - PostgreSQL: Supabase (500MB free)
   - Redis: Upstash (10K commands/day free)
   - Deploy only bot on Railway

2. **Switch to Fly.io:**
   - Free tier: 3 VMs
   - Use Supabase + Upstash
   - Total cost: $0

3. **Self-host:**
   - VPS: $4-5/month (Hetzner, Contabo)
   - Full control, no limits

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

**Last Updated**: 2025-01-03
**Maintained by**: [@valpere](https://github.com/valpere)
