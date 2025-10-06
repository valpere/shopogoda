# Vercel Deployment Guide

This guide provides detailed step-by-step instructions for deploying ShoPogoda bot to Vercel serverless platform.

## Prerequisites

- Vercel account (https://vercel.com)
- Vercel CLI installed: `npm install -g vercel`
- Bot working locally (verified with `make run`)
- Telegram Bot Token from @BotFather
- OpenWeather API Key from https://openweathermap.org
- PostgreSQL database (e.g., Supabase)

## Architecture Overview

Vercel deployment uses:
- **Serverless Functions**: Go handlers in `api/` directory
- **Webhook Mode**: Telegram sends updates via HTTP POST
- **Stateless Design**: Each request is independent (cold start initialization)
- **Connection Pooling**: Optimized for serverless (5 max connections, 2 idle)
- **Timeout**: 9-second processing limit (Vercel has 10s max)

## Step 1: Prepare Environment Variables

### Required Variables

You need to set these environment variables in Vercel:

1. **TELEGRAM_BOT_TOKEN**
   - Get from @BotFather on Telegram
   - Format: `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`
   - **Critical**: No trailing newlines or spaces

2. **DATABASE_URL**
   - PostgreSQL connection string
   - Format: `postgresql://user:password@host:port/dbname?sslmode=require`
   - Example (Supabase): `postgresql://postgres.xxx:password@aws-0-eu-central-1.pooler.supabase.com:6543/postgres?sslmode=require`

3. **OPENWEATHER_API_KEY**
   - Get from https://openweathermap.org/api
   - Format: 32-character hex string
   - **Critical**: No trailing newlines or spaces

### Optional Variables

4. **REDIS_URL** (optional, bot works without cache)
   - Redis connection string
   - Format: `redis://user:password@host:port`
   - Note: Standard Redis may not work well on Vercel (consider Upstash REST API)

5. **BOT_DEBUG** (optional, for debugging)
   - Set to `true` to enable debug logging
   - Default: `false`

6. **LOG_FORMAT** (optional)
   - Set to `json` for JSON logging
   - Default: console format with colors

### Get Your Environment Variable Values

From your local `.env` file:

```bash
# View your current values (without exposing full secrets)
echo "TELEGRAM_BOT_TOKEN length: $(echo -n "$TELEGRAM_BOT_TOKEN" | wc -c) chars"
echo "DATABASE_URL host: $(echo "$DATABASE_URL" | grep -oP '(?<=@)[^:]+' || echo 'not set')"
echo "OPENWEATHER_API_KEY length: $(echo -n "$OPENWEATHER_API_KEY" | wc -c) chars"
```

## Step 2: Configure Vercel Project

### Initial Vercel Setup

If this is your first deployment:

```bash
# Login to Vercel
vercel login

# Link project (run from project root)
vercel link
# - Choose your scope (personal account or team)
# - Link to existing project? No (first time) or Yes (re-deploying)
# - Project name: shopogoda
```

### Set Environment Variables in Vercel

**Option A: Via Vercel CLI (Recommended)**

```bash
# Navigate to project root
cd /home/val/wrk/projects/telegram_bot/shopogoda

# Set TELEGRAM_BOT_TOKEN
# IMPORTANT: Use echo -n to avoid newlines
echo -n "YOUR_TELEGRAM_BOT_TOKEN" | vercel env add TELEGRAM_BOT_TOKEN production

# Set DATABASE_URL
echo -n "YOUR_DATABASE_URL" | vercel env add DATABASE_URL production

# Set OPENWEATHER_API_KEY
echo -n "YOUR_OPENWEATHER_API_KEY" | vercel env add OPENWEATHER_API_KEY production

# Optional: Set REDIS_URL
echo -n "YOUR_REDIS_URL" | vercel env add REDIS_URL production
```

**Option B: Via Vercel Dashboard**

1. Go to https://vercel.com/dashboard
2. Select your project `shopogoda`
3. Go to Settings → Environment Variables
4. Add each variable:
   - Variable Name: `TELEGRAM_BOT_TOKEN`
   - Value: Your bot token (paste carefully, no extra spaces)
   - Environment: Production
   - Click "Save"
5. Repeat for `DATABASE_URL` and `OPENWEATHER_API_KEY`

### Verify Environment Variables

```bash
# List all environment variables (values are hidden)
vercel env ls

# Expected output:
# Environment Variables
#   Name                    Environments
#   TELEGRAM_BOT_TOKEN      Production
#   DATABASE_URL            Production
#   OPENWEATHER_API_KEY     Production
```

## Step 3: Verify Configuration Files

### Check vercel.json

Your `vercel.json` should look like this:

```json
{
  "version": 2,
  "builds": [
    {
      "src": "api/webhook.go",
      "use": "@vercel/go"
    },
    {
      "src": "api/health.go",
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

### Check go.mod GORM Version

**Critical**: Ensure GORM is v1.25.6, not v1.30.0 (v1.30.0 has migration bug)

```bash
grep "gorm.io/gorm" go.mod
# Should show: gorm.io/gorm v1.25.6
```

If it shows v1.30.0, downgrade:

```bash
go get gorm.io/gorm@v1.25.6
go mod tidy
```

### Verify Webhook Handler

Check that `api/webhook.go` exists and has:
- `Handler(w http.ResponseWriter, r *http.Request)` function
- Synchronous update processing (not goroutine)
- Proper DATABASE_URL environment variable handling
- GORM AutoMigrate on initialization

## Step 4: Deploy to Vercel

### Preview Deployment (Optional)

Test deployment without affecting production:

```bash
# Deploy to preview environment
vercel

# Wait for deployment to complete
# Note the preview URL: https://shopogoda-xyz.vercel.app

# Test health endpoint
curl https://shopogoda-xyz.vercel.app/health

# Expected response:
# {"status":"healthy"}
```

### Production Deployment

```bash
# Deploy to production
vercel --prod

# Expected output:
# Vercel CLI 32.0.1
# Deploying to production
# Building...
# Deploying... (may take 1-2 minutes)
# ✓ Production: https://shopogoda.vercel.app [1m]
```

### Verify Deployment

```bash
# Check deployment status
vercel ls

# Test health endpoint
curl https://shopogoda.vercel.app/health

# Expected: {"status":"healthy"}

# Check webhook endpoint (should return 405 Method Not Allowed for GET)
curl https://shopogoda.vercel.app/webhook

# Expected: Method Not Allowed (this is correct - webhook only accepts POST)
```

### View Deployment Logs

```bash
# View real-time logs
vercel logs shopogoda --follow

# Or view recent logs
vercel logs shopogoda
```

## Step 5: Configure Telegram Webhook

### Set Webhook URL

```bash
# Set webhook to your Vercel deployment
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook?url=https://shopogoda.vercel.app/api/webhook&drop_pending_updates=true"

# Expected response:
# {"ok":true,"result":true,"description":"Webhook was set"}
```

### Verify Webhook Configuration

```bash
# Check webhook info
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getWebhookInfo"

# Expected response should include:
# {
#   "ok": true,
#   "result": {
#     "url": "https://shopogoda.vercel.app/api/webhook",
#     "has_custom_certificate": false,
#     "pending_update_count": 0,
#     "max_connections": 40
#   }
# }
```

### Alternative: Use GitHub CLI (if configured)

```bash
# Set webhook using gh api
gh api --method POST "/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  -f url='https://shopogoda.vercel.app/api/webhook' \
  -F drop_pending_updates=true
```

## Step 6: Test Bot on Telegram

### Test Basic Commands

1. Open Telegram and find your bot (e.g., @shopogoda_dev_bot)

2. Send `/start` command
   - Expected: Welcome message with inline keyboard buttons

3. Send `/help` command
   - Expected: List of available commands

4. Send `/weather` command (if you have location set)
   - Expected: Current weather for your location

### Monitor Logs During Testing

In a separate terminal, watch Vercel logs:

```bash
vercel logs shopogoda --follow
```

When you send a command, you should see:
- Request received
- Update processing
- Database queries
- Response sent

### Troubleshooting Tests

**If bot doesn't respond:**

1. Check webhook is set:
   ```bash
   curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getWebhookInfo"
   ```

2. Check Vercel logs for errors:
   ```bash
   vercel logs shopogoda
   ```

3. Test webhook endpoint directly:
   ```bash
   curl -X POST https://shopogoda.vercel.app/api/webhook \
     -H "Content-Type: application/json" \
     -d '{"update_id":1,"message":{"message_id":1,"from":{"id":123,"is_bot":false,"first_name":"Test"},"chat":{"id":123,"type":"private"},"date":1234567890,"text":"/start"}}'
   ```

4. Check environment variables are set:
   ```bash
   vercel env ls
   ```

## Common Issues and Solutions

### Issue 1: Bot Doesn't Respond

**Symptoms:**
- Commands sent to bot receive no response
- Webhook shows as set but no logs in Vercel

**Diagnosis:**
```bash
# Check webhook status
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getWebhookInfo"

# Check recent errors
vercel logs shopogoda | grep -i error
```

**Solutions:**
1. Verify webhook URL is correct (includes `/api/webhook` path)
2. Check TELEGRAM_BOT_TOKEN has no newlines: `echo -n "$TOKEN" | wc -c`
3. Redeploy: `vercel --prod`
4. Re-set webhook with `drop_pending_updates=true`

### Issue 2: "insufficient arguments" GORM Error

**Symptoms:**
- Deployment succeeds but bot crashes on startup
- Logs show: `insufficient arguments` from GORM

**Solution:**
```bash
# Downgrade GORM to v1.25.6
go get gorm.io/gorm@v1.25.6
go mod tidy

# Redeploy
vercel --prod
```

### Issue 3: Database Connection Timeout

**Symptoms:**
- Logs show: `failed to connect to database`
- Error: `connection timeout`

**Solutions:**
1. Check DATABASE_URL is correct:
   ```bash
   vercel env ls
   ```
2. Verify PostgreSQL allows connections from Vercel IPs
3. Use connection pooler (port 6543 for Supabase, not 5432)
4. Ensure `?sslmode=require` is in connection string

### Issue 4: Cold Start Timeout

**Symptoms:**
- First request after idle period times out
- Subsequent requests work fine

**Solutions:**
1. This is normal for serverless - cold starts take 2-5 seconds
2. Consider keeping bot "warm" with periodic health checks:
   ```bash
   # Add to cron (every 5 minutes)
   */5 * * * * curl https://shopogoda.vercel.app/health
   ```
3. For production, consider using Vercel's "Cron Jobs" feature

### Issue 5: Environment Variable Not Set

**Symptoms:**
- Error: `TELEGRAM_BOT_TOKEN is required`
- Error: `OPENWEATHER_API_KEY is required`

**Solution:**
```bash
# Check if variable exists
vercel env ls

# Remove and re-add (ensures no newlines)
vercel env rm TELEGRAM_BOT_TOKEN production
echo -n "YOUR_TOKEN_HERE" | vercel env add TELEGRAM_BOT_TOKEN production

# Redeploy
vercel --prod
```

### Issue 6: Webhook Not Persisting

**Symptoms:**
- Webhook shows as set immediately after setWebhook
- After a few minutes, webhook is null again

**Solutions:**
1. Verify bot token is correct
2. Ensure webhook URL is publicly accessible
3. Check Telegram can reach your endpoint:
   ```bash
   curl -X POST https://shopogoda.vercel.app/api/webhook \
     -H "Content-Type: application/json" \
     -d '{"test": true}'
   # Should return 200 OK
   ```
4. Re-set webhook with certificate parameter if using HTTPS

## Monitoring and Maintenance

### Daily Monitoring

```bash
# Check deployment health
curl https://shopogoda.vercel.app/health

# View recent logs
vercel logs shopogoda

# Check webhook status
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getWebhookInfo"
```

### Monthly Maintenance

1. Check for dependency updates:
   ```bash
   go list -u -m all
   ```

2. Review Vercel usage:
   - Visit https://vercel.com/dashboard
   - Check function execution count
   - Monitor bandwidth usage

3. Verify database performance:
   - Check connection pool usage
   - Review slow queries

### Rollback Procedure

If a deployment breaks the bot:

```bash
# List recent deployments
vercel ls

# Promote a previous deployment to production
vercel promote <deployment-url>

# Or rollback via dashboard:
# 1. Go to https://vercel.com/dashboard
# 2. Select project → Deployments
# 3. Find working deployment → Promote to Production
```

## Performance Optimization

### Connection Pooling

Current settings (optimized for serverless):
```go
sqlDB.SetMaxOpenConns(5)                  // Small pool for serverless
sqlDB.SetMaxIdleConns(2)                  // Minimal idle connections
sqlDB.SetConnMaxLifetime(5 * time.Minute) // Short lifetime
```

### Caching Strategy

Without Redis:
- Each request queries database
- Weather API calls are made on every request
- Acceptable for low-traffic bots

With Redis (optional):
- Add Upstash Redis for free tier
- Set REDIS_URL environment variable
- Implement caching in services

### Cold Start Optimization

Current cold start: ~2-5 seconds
- Database migration check: ~1s
- Bot initialization: ~1s
- First request processing: ~1s

To improve:
1. Keep functions warm (periodic health checks)
2. Optimize imports (remove unused packages)
3. Consider connection pooling service (PgBouncer)

## Security Best Practices

### Environment Variables

- ✅ Never commit `.env` to git
- ✅ Use `echo -n` to avoid newlines in secrets
- ✅ Rotate secrets regularly (every 90 days)
- ✅ Use different tokens for dev/production

### Database Access

- ✅ Use connection pooler (port 6543)
- ✅ Require SSL: `?sslmode=require`
- ✅ Limit connection lifetime
- ✅ Use least-privilege database user

### Telegram Security

- ✅ Verify webhook requests (optional: add secret token)
- ✅ Validate user input
- ✅ Rate limit commands (implemented in middleware)
- ✅ Monitor for abuse

## Cost Estimation

### Vercel Free Tier Limits

- **Function Executions**: 100GB-hrs/month (≈ 100,000 requests)
- **Function Duration**: 10 seconds max per request
- **Bandwidth**: 100GB/month

### Typical Usage (Small Bot)

- **Daily Active Users**: 100
- **Average Commands/User/Day**: 10
- **Monthly Executions**: 30,000 (well within free tier)

### When to Upgrade

Consider Vercel Pro ($20/month) when:
- More than 3,000 users
- More than 100,000 requests/month
- Need longer function duration (60s)
- Need more bandwidth

## Alternative Deployment Options

If Vercel doesn't meet your needs:

1. **Fly.io** (current setup exists in `fly.toml`)
   - Pros: Always-on, no cold starts, better for scheduled tasks
   - Cons: Requires Dockerfile, more expensive
   - Guide: See `docs/FLY_DEPLOYMENT.md`

2. **Railway**
   - Pros: Easy PostgreSQL integration, GitHub auto-deploy
   - Cons: Paid service only, no free tier
   - Setup: Connect GitHub repo, add environment variables

3. **AWS Lambda**
   - Pros: Massive free tier, good for high traffic
   - Cons: More complex setup, requires API Gateway
   - Setup: Use AWS SAM or Serverless Framework

4. **Self-hosted VPS**
   - Pros: Full control, predictable costs
   - Cons: Requires maintenance, security updates
   - Setup: Use Docker Compose from repository

## Conclusion

You now have a production-ready Telegram bot deployed on Vercel serverless platform. The bot will:

- ✅ Start cold on first request (~2-5s)
- ✅ Process subsequent requests quickly (~100-500ms)
- ✅ Automatically scale with traffic
- ✅ Run migrations on first startup
- ✅ Handle commands via Telegram webhook

### Next Steps

1. Monitor logs for the first 24 hours
2. Test all commands thoroughly
3. Set up monitoring (consider Vercel integrations)
4. Add cron job to keep function warm
5. Configure alerts for errors

### Getting Help

- **Vercel Docs**: https://vercel.com/docs
- **gotgbot Docs**: https://pkg.go.dev/github.com/PaulSonOfLars/gotgbot/v2
- **Project Issues**: Check `docs/TROUBLESHOOTING.md`
- **Telegram Bot API**: https://core.telegram.org/bots/api

---

**Document Version**: 1.0
**Last Updated**: 2025-10-06
**Tested With**: Vercel CLI 32.0.1, Go 1.24.6, gotgbot v2.0.0-rc.25
