# Replit Deployment Guide

Complete step-by-step guide for deploying ShoPogoda Telegram bot on Replit.

## Why Replit?

Replit is an excellent choice for deploying Telegram bots, especially for beginners and small-to-medium scale deployments.

### Advantages

✅ **Beginner-Friendly**: Web-based IDE with visual interface
✅ **Built-in Database**: PostgreSQL included (no external setup needed)
✅ **Always-On Option**: No cold starts with paid plan
✅ **Simple Setup**: No Docker, no complex configuration
✅ **Environment Variables**: Easy secrets management via UI
✅ **GitHub Integration**: Auto-deploy on git push
✅ **Free Tier**: Test before committing to paid plan
✅ **Terminal Access**: Debug directly in browser
✅ **Collaborative**: Share Repl with team members

### Disadvantages

❌ **Free Tier Limitations**: Bot sleeps after ~1 hour of inactivity
❌ **Cost**: $20/month for always-on (more expensive than some alternatives)
❌ **Resource Limits**: Less control over CPU/memory allocation
❌ **Vendor Lock-in**: Replit-specific configuration files

### When to Choose Replit

**Choose Replit if:**
- You're new to deployment and want simplicity
- You want built-in PostgreSQL database
- You need always-on capability (paid plan)
- You prefer visual interface over command line
- You want quick prototyping and testing

**Choose alternatives if:**
- You need serverless (Vercel)
- You want more control (Fly.io, VPS)
- You have high traffic (AWS Lambda)
- You're budget-conscious (self-hosted VPS ~$5/month)

## Prerequisites

### Required
- Replit account (https://replit.com - free signup)
- Telegram Bot Token from @BotFather
- OpenWeather API Key from https://openweathermap.org/api

### Optional
- GitHub account (for auto-deploy)
- External PostgreSQL database (if not using Replit's built-in DB)
- Redis instance (optional, bot works without cache)

## Step 1: Create Replit Account

### Sign Up

1. Go to https://replit.com
2. Click "Sign up"
3. Choose signup method:
   - GitHub (recommended - enables auto-deploy)
   - Google
   - Email

4. Verify your email if required

### Understand Replit Plans

**Free Plan ($0/month):**
- ✅ Unlimited public Repls
- ✅ Basic compute resources
- ❌ Repls sleep after ~1 hour inactivity
- ❌ No always-on capability
- ❌ Limited outbound bandwidth

**Hacker Plan ($7/month):**
- ✅ Private Repls
- ✅ More compute resources
- ❌ Repls still sleep
- ❌ No always-on capability

**Pro Plan ($20/month):**
- ✅ All Hacker features
- ✅ Always-on Repls (recommended for bots)
- ✅ Boosted resources
- ✅ Custom domains

**For Production Bots**: Pro Plan ($20/month) is recommended for 24/7 uptime.

## Step 2: Import Project from GitHub

### Option A: Import from GitHub (Recommended)

1. Click "Create Repl" button (top left)
2. Select "Import from GitHub" tab
3. Enter repository URL:
   ```
   https://github.com/valpere/shopogoda
   ```
4. Select language: **Go**
5. Click "Import from GitHub"
6. Wait for import to complete (~30 seconds)

### Option B: Create from Template (Alternative)

1. Click "Create Repl"
2. Select "Go" as language
3. Name your Repl: `shopogoda`
4. Click "Create Repl"
5. In terminal, clone repository:
   ```bash
   git clone https://github.com/valpere/shopogoda .
   ```

### Verify Import

After import completes, you should see:
- File explorer on left with project files
- `cmd/`, `internal/`, `api/` directories visible
- `go.mod`, `Makefile`, `README.md` in root

## Step 3: Configure Replit Environment

### Verify Configuration Files

Replit should detect these files automatically:

**`.replit`** - Run configuration (already in repository):
```toml
run = "make build && ./bin/shopogoda"

[nix]
channel = "stable-23_11"

[deployment]
run = ["sh", "-c", "make build && ./bin/shopogoda"]
deploymentTarget = "gce"

[[ports]]
localPort = 8080
externalPort = 80
```

**`replit.nix`** - Dependencies (already in repository):
```nix
{ pkgs }: {
  deps = [
    pkgs.go_1_21
    pkgs.postgresql
    pkgs.redis
    pkgs.gnumake
    pkgs.gcc
  ];
}
```

### If Files Are Missing

If `.replit` or `replit.nix` are missing, create them:

1. Click "Files" icon (folder icon in left sidebar)
2. Click three dots (⋮) → "New file"
3. Name: `.replit`
4. Paste content from above
5. Repeat for `replit.nix`

## Step 4: Set Up Environment Variables (Secrets)

### Access Secrets Manager

1. Look for "Secrets" icon in left sidebar (🔒 lock icon)
   - Or click "Tools" → "Secrets"
2. Click "Secrets" to open secrets panel

### Add Required Secrets

Add these secrets one by one:

#### 1. TELEGRAM_BOT_TOKEN

```
Key: TELEGRAM_BOT_TOKEN
Value: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz
```

**How to get:**
1. Open Telegram and search for @BotFather
2. Send `/newbot` command
3. Follow prompts to create bot
4. Copy the token provided
5. **Important**: Paste exactly, no extra spaces or newlines

#### 2. OPENWEATHER_API_KEY

```
Key: OPENWEATHER_API_KEY
Value: your_32_character_api_key_here
```

**How to get:**
1. Go to https://openweathermap.org/api
2. Sign up for free account
3. Go to "API keys" section
4. Copy your API key
5. **Important**: No quotes, just the raw key

#### 3. DATABASE_URL

**Option A: Use Replit's Built-in PostgreSQL (Recommended)**

```
Key: DATABASE_URL
Value: (will be set after enabling database)
```

Skip this for now - we'll set it in Step 5.

**Option B: Use External Database (Supabase, etc.)**

```
Key: DATABASE_URL
Value: postgresql://user:password@host:port/dbname?sslmode=require
```

Example (Supabase):
```
postgresql://postgres.xxx:password@aws-0-eu-central-1.pooler.supabase.com:6543/postgres?sslmode=require
```

#### 4. BOT_MODE (Required for Polling)

```
Key: BOT_MODE
Value: polling
```

**Important**: Replit works best with polling mode (not webhook) because:
- No need for public HTTPS URL
- Simpler setup
- Works on free tier

#### Optional Secrets

**REDIS_URL** (optional - bot works without cache):
```
Key: REDIS_URL
Value: redis://host:port
```

**BOT_DEBUG** (for debugging):
```
Key: BOT_DEBUG
Value: true
```

**LOG_FORMAT** (for JSON logs):
```
Key: LOG_FORMAT
Value: json
```

### Verify Secrets

After adding all secrets, you should see:
- ✅ TELEGRAM_BOT_TOKEN
- ✅ OPENWEATHER_API_KEY
- ✅ DATABASE_URL (or pending for Step 5)
- ✅ BOT_MODE

## Step 5: Set Up Replit Database (Built-in PostgreSQL)

### Enable PostgreSQL

1. Click "Database" icon in left sidebar (🗄️ database icon)
2. Click "Enable PostgreSQL"
3. Wait for database to provision (~10-30 seconds)
4. Once enabled, you'll see connection details

### Get Connection String

1. In Database panel, look for "Connection String"
2. Click "Copy" button
3. Format will be:
   ```
   postgresql://username:password@db.postgres.repl.co:5432/dbname
   ```

### Add to Secrets

1. Go back to "Secrets" panel
2. Find `DATABASE_URL` secret
3. Paste the connection string you copied
4. Click "Save"

### Alternative: Use External Database

If you prefer external PostgreSQL (Supabase, etc.):

**Supabase Setup:**
1. Go to https://supabase.com
2. Create project
3. Go to Settings → Database
4. Copy "Connection string" (Transaction mode)
5. Replace `[YOUR-PASSWORD]` with your actual password
6. Add `?sslmode=require` at the end
7. Paste into DATABASE_URL secret

**Neon Setup:**
1. Go to https://neon.tech
2. Create project
3. Copy connection string
4. Paste into DATABASE_URL secret

## Step 6: Configure Bot for Polling Mode

The bot needs to run in polling mode on Replit (not webhook). Let's verify the configuration:

### Check Main Entry Point

Open `cmd/bot/main.go` and verify it supports polling mode.

If the bot only supports webhook mode, we need to add polling support:

1. Click "Files" → `cmd/bot/main.go`
2. Find the bot startup logic
3. Add polling mode check:

```go
// After bot initialization, add this:

// Determine bot mode (polling vs webhook)
botMode := os.Getenv("BOT_MODE")
if botMode == "polling" || cfg.Bot.WebhookURL == "" {
    logger.Info().Msg("Starting bot in polling mode")

    // Start polling
    err = updater.StartPolling(bot, &ext.PollingOpts{
        DropPendingUpdates: true,
        GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
            Timeout: 10,
            RequestOpts: &gotgbot.RequestOpts{
                Timeout: time.Second * 10,
            },
        },
    })
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to start polling")
    }

    logger.Info().Msg("Bot started successfully in polling mode")
    logger.Info().Str("username", bot.User.Username).Msg("Bot info")

    // Block until interrupt signal
    updater.Idle()
} else {
    // Webhook mode (existing code)
    logger.Info().Msg("Starting bot in webhook mode")
    // ... existing webhook code ...
}
```

### Verify Config Loading

Ensure `internal/config/config.go` loads from environment variables:

```go
type Config struct {
    Bot struct {
        Token      string `mapstructure:"token"`
        WebhookURL string `mapstructure:"webhook_url"`
        Mode       string `mapstructure:"mode"` // polling or webhook
        Debug      bool   `mapstructure:"debug"`
    } `mapstructure:"bot"`
    // ... rest of config
}
```

## Step 7: Run the Bot

### First Run

1. Click the green "▶ Run" button at top
2. Watch the console output
3. You should see:
   ```
   Starting ShoPogoda v0.1.0
   Connecting to database...
   Running migrations...
   Loaded translations...
   Starting bot in polling mode...
   Bot started successfully in polling mode
   Bot info: @your_bot_name
   ```

### Common First-Run Issues

**Issue: "TELEGRAM_BOT_TOKEN is required"**
- Solution: Check Secrets panel, ensure token is set correctly

**Issue: "failed to connect to database"**
- Solution: Verify DATABASE_URL secret is set with correct connection string

**Issue: "insufficient arguments" (GORM error)**
- Solution: Check `go.mod` has `gorm.io/gorm v1.25.6` (not v1.30.0)
- Fix: In terminal, run:
  ```bash
  go get gorm.io/gorm@v1.25.6
  go mod tidy
  ```

**Issue: "package not found"**
- Solution: Install dependencies:
  ```bash
  go mod download
  go mod tidy
  ```

### Stop the Bot

- Click the "Stop" button (⏹️ square icon) at top
- Or press Ctrl+C in console

## Step 8: Test the Bot

### Find Your Bot on Telegram

1. Open Telegram app
2. Search for your bot username (e.g., @shopogoda_dev_bot)
3. Click on the bot to open chat

### Test Basic Commands

Send these commands to verify bot works:

**1. Test /start command:**
```
/start
```
Expected: Welcome message with inline keyboard buttons

**2. Test /help command:**
```
/help
```
Expected: List of available commands

**3. Test /version command:**
```
/version
```
Expected: Bot version information

**4. Test /setlocation command:**
```
/setlocation
```
Expected: Instructions to set location or button to share location

**5. Test /weather command (after setting location):**
```
/weather
```
Expected: Current weather for your location

### Check Replit Console

While testing, watch the Replit console for:
- ✅ "Received update" messages
- ✅ Command processing logs
- ✅ Database queries (if debug enabled)
- ❌ Any error messages

### Debug Mode

To see detailed logs:

1. Go to Secrets
2. Add or update:
   ```
   Key: BOT_DEBUG
   Value: true
   ```
3. Restart bot (Stop and Run again)
4. Now you'll see detailed logs for every update

## Step 9: Keep Bot Running (Free Tier)

### The Problem

Free Replit accounts have bot sleep after ~1 hour of inactivity.

### Solution 1: Upgrade to Pro ($20/month)

**Recommended for production bots**

1. Click your profile icon (top right)
2. Select "Upgrade to Pro"
3. Choose Pro plan ($20/month)
4. Enable "Always On" for your Repl:
   - Click "Deploy" button
   - Select "Autoscale Deployment" or "Reserved VM"
   - Your bot will run 24/7 without sleep

### Solution 2: Use UptimeRobot (Free Tier Workaround)

**For testing/development only**

1. Get your Repl's public URL:
   - Look at the webview panel (usually top right)
   - URL format: `https://shopogoda.username.repl.co`
   - Note: The bot must expose an HTTP endpoint for this to work

2. Create health endpoint (if not exists):
   - Add to `api/health.go` (should already exist):
     ```go
     package handler

     import (
         "encoding/json"
         "net/http"
     )

     func Health(w http.ResponseWriter, r *http.Request) {
         w.Header().Set("Content-Type", "application/json")
         json.NewEncoder(w).Encode(map[string]string{
             "status": "healthy",
         })
     }
     ```

3. Add HTTP server to main.go (if polling mode):
   ```go
   // Start health check server for UptimeRobot
   go func() {
       http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
           w.WriteHeader(http.StatusOK)
           w.Write([]byte("OK"))
       })
       if err := http.ListenAndServe(":8080", nil); err != nil {
           logger.Error().Err(err).Msg("Health server failed")
       }
   }()
   ```

4. Sign up at https://uptimerobot.com (free)

5. Add new monitor:
   - Monitor Type: HTTP(s)
   - Friendly Name: ShoPogoda Bot
   - URL: `https://shopogoda.username.repl.co/health`
   - Monitoring Interval: 5 minutes
   - Click "Create Monitor"

**Note**: Free Repls still sleep after extended inactivity. This only extends uptime, not guaranteed 24/7.

### Solution 3: Periodic Git Push (Auto-Wake)

Replit wakes up when you push to connected GitHub repo:

1. Set up GitHub auto-deploy:
   - In Repl settings, connect to GitHub
   - Enable "Run on every commit"

2. Create a cron job (on your computer/server) to push empty commit:
   ```bash
   #!/bin/bash
   # keep-repl-alive.sh
   cd /path/to/local/repo
   git commit --allow-empty -m "Keep Repl alive"
   git push
   ```

3. Add to crontab (every hour):
   ```
   0 * * * * /path/to/keep-repl-alive.sh
   ```

**Note**: This is a workaround, not reliable for production.

## Step 10: Enable Always-On (Production Setup)

### For 24/7 Production Bot

1. Upgrade to Replit Pro ($20/month)
2. Click "Deploy" button (top right)
3. Choose deployment type:

**Option A: Autoscale Deployment (Recommended)**
   - Scales automatically with load
   - Pay for what you use
   - Better for variable traffic
   - Click "Configure Autoscale"
   - Set min/max instances
   - Click "Deploy"

**Option B: Reserved VM**
   - Dedicated VM for your Repl
   - Fixed cost
   - Consistent performance
   - Better for consistent traffic
   - Click "Configure Reserved VM"
   - Choose VM size (Small recommended: 0.5 vCPU, 512 MB RAM)
   - Click "Deploy"

### Verify Always-On Status

1. Go to your Repl
2. Look for "Always On" badge (green checkmark)
3. Bot should show as "Running" even when you close browser

### Monitor Deployment

1. Click "Deployments" tab
2. View logs, metrics, and status
3. Check uptime and resource usage

## Monitoring and Maintenance

### View Logs

**Console Logs (Real-time):**
1. Open your Repl
2. View console panel (bottom)
3. Logs appear as bot processes updates

**Deployment Logs (Historical):**
1. Click "Deployments" tab
2. Select deployment
3. Click "Logs"
4. Filter by date/severity

### Monitor Bot Health

**Check Bot Status:**
```bash
# In Replit shell
curl localhost:8080/health
```

**Check Database Connection:**
```bash
# In Replit shell
psql $DATABASE_URL -c "SELECT version();"
```

**Check Bot Info on Telegram:**
```bash
# In Replit shell
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getMe"
```

### Performance Monitoring

1. Click "Resources" tab
2. View CPU, memory, disk usage
3. Adjust VM size if needed

### Database Management

**Access Replit Database:**
1. Click "Database" icon
2. View tables and data in GUI
3. Or use psql in shell:
   ```bash
   psql $DATABASE_URL
   ```

**Common Database Commands:**
```sql
-- List tables
\dt

-- Count users
SELECT COUNT(*) FROM users;

-- View recent weather data
SELECT * FROM weather_data ORDER BY created_at DESC LIMIT 10;

-- Check subscriptions
SELECT user_id, frequency, is_active FROM subscriptions;
```

## Troubleshooting

### Bot Not Responding to Commands

**Symptoms:**
- Send /start command
- No response from bot

**Diagnosis:**
1. Check console for errors
2. Verify bot is running (see "Running" status)
3. Check Telegram bot token is correct

**Solutions:**

1. **Verify bot is running:**
   ```bash
   # Should show bot process
   ps aux | grep shopogoda
   ```

2. **Check secrets:**
   - Go to Secrets panel
   - Verify TELEGRAM_BOT_TOKEN is set
   - Ensure no extra spaces or newlines

3. **Restart bot:**
   - Click Stop button
   - Click Run button
   - Watch console for startup messages

4. **Check polling mode:**
   - Verify BOT_MODE=polling in secrets
   - Check logs for "Starting bot in polling mode"

### Database Connection Errors

**Symptoms:**
- Error: "failed to connect to database"
- Error: "connection refused"

**Solutions:**

1. **Verify DATABASE_URL:**
   ```bash
   # In shell
   echo $DATABASE_URL
   ```

2. **Test connection:**
   ```bash
   psql $DATABASE_URL -c "SELECT 1;"
   ```

3. **Re-enable database:**
   - Click Database icon
   - If disconnected, click "Enable PostgreSQL" again
   - Copy new connection string
   - Update DATABASE_URL secret

4. **Check database is running:**
   - Database icon should show green indicator
   - If red, click to restart

### GORM Migration Errors

**Symptoms:**
- Error: "insufficient arguments"
- Error: "SELECT * FROM users LIMIT 1"

**Cause:**
GORM v1.30.0 has a bug with AutoMigrate

**Solution:**

1. Open shell (click "Shell" in bottom panel)
2. Downgrade GORM:
   ```bash
   go get gorm.io/gorm@v1.25.6
   go mod tidy
   ```
3. Restart bot

### Out of Memory Errors

**Symptoms:**
- Bot crashes randomly
- Error: "killed" or "out of memory"

**Solutions:**

1. **Upgrade VM size:**
   - Go to Deployments → Configure
   - Choose larger VM (Medium: 1 vCPU, 1 GB RAM)

2. **Optimize connection pool:**
   - Edit `internal/database/database.go`
   - Reduce MaxOpenConns:
     ```go
     sqlDB.SetMaxOpenConns(3)  // Lower for Replit
     sqlDB.SetMaxIdleConns(1)
     ```

3. **Disable Redis cache** (if enabled):
   - Remove REDIS_URL from secrets
   - Bot uses less memory without cache

### Repl Keeps Sleeping (Free Tier)

**Solutions:**

1. **Upgrade to Pro** ($20/month) - Recommended
2. **Use UptimeRobot** - Extends uptime but not 24/7
3. **Manual wake-up** - Open Repl in browser periodically

### Bot Slow to Respond

**Causes:**
- Cold start after sleep (free tier)
- Database query slowness
- External API calls (weather API)

**Solutions:**

1. **Enable Always-On** (Pro plan)
2. **Add Redis caching:**
   - Set up external Redis (Upstash free tier)
   - Add REDIS_URL to secrets
   - Bot caches weather data (10 min TTL)

3. **Optimize database indexes:**
   ```sql
   -- Add indexes if missing
   CREATE INDEX IF NOT EXISTS idx_users_id ON users(id);
   CREATE INDEX IF NOT EXISTS idx_weather_data_user_id ON weather_data(user_id);
   ```

## Security Best Practices

### Protect Secrets

✅ **Never commit secrets to git**
- Use Replit Secrets panel only
- Don't hardcode tokens in code
- Don't share secrets in chat/email

✅ **Rotate secrets regularly**
- Change bot token every 90 days
- Use @BotFather's /revoke to get new token

✅ **Use different bots for dev/prod**
- Dev bot: @shopogoda_dev_bot
- Prod bot: @shopogoda_bot
- Different tokens for each

### Database Security

✅ **Use connection pooling**
```go
sqlDB.SetMaxOpenConns(5)
sqlDB.SetMaxIdleConns(2)
sqlDB.SetConnMaxLifetime(5 * time.Minute)
```

✅ **Validate user input**
- Never trust user input
- Sanitize location names
- Validate numeric inputs

✅ **Use prepared statements**
- GORM does this automatically
- Prevents SQL injection

### Bot Security

✅ **Implement rate limiting**
- Already implemented in middleware
- 10 requests/minute per user

✅ **Validate Telegram updates**
- Check user IDs are valid
- Verify message structure

✅ **Monitor for abuse**
- Log suspicious activity
- Block malicious users

## Cost Analysis

### Free Tier

**What you get:**
- ✅ Unlimited public Repls
- ✅ Basic compute (0.2 vCPU, 256 MB RAM)
- ✅ PostgreSQL database
- ❌ Bot sleeps after ~1 hour inactivity
- ❌ No guaranteed uptime

**Best for:**
- Development and testing
- Learning and prototyping
- Low-traffic personal bots

**Estimated capacity:**
- ~10-50 users
- Intermittent usage
- Not for production

### Pro Plan ($20/month)

**What you get:**
- ✅ Always-on capability
- ✅ Boosted compute (0.5 vCPU, 512 MB RAM)
- ✅ Private Repls
- ✅ Priority support
- ✅ Custom domains

**Best for:**
- Production bots
- 24/7 availability
- 100-1000 users

**Estimated capacity:**
- ~100-500 active users
- ~10,000 messages/day
- Continuous operation

### Cost Comparison

| Feature | Replit Free | Replit Pro | Vercel Free | Fly.io | VPS |
|---------|-------------|------------|-------------|---------|-----|
| Cost | $0 | $20/month | $0 | ~$5/month | ~$5-20/month |
| Uptime | Intermittent | 24/7 | 24/7 | 24/7 | 24/7 |
| Database | Included | Included | External | Add-on | Self-managed |
| Setup | Easy | Easy | Medium | Hard | Hard |
| Cold Starts | Yes | No | Yes (2-5s) | No | No |

### When to Upgrade

**Upgrade from Free to Pro when:**
- Users complain bot is offline
- You need 24/7 availability
- Bot is used for business/critical tasks
- You have >50 active users

**Migrate from Replit when:**
- Cost exceeds $20/month value
- Need more control (Fly.io, VPS)
- Need serverless at scale (Vercel, AWS Lambda)
- Want cheaper hosting (VPS ~$5/month)

## Advanced Configuration

### Custom Domain

**Requirements:**
- Replit Pro plan
- Domain name purchased separately

**Setup:**
1. Go to your Repl
2. Click "Deployments" → "Custom Domain"
3. Enter your domain: `bot.yourdomain.com`
4. Add DNS records as instructed
5. Wait for propagation (5-30 minutes)

### GitHub Auto-Deploy

**Setup:**
1. Click Repl settings (gear icon)
2. Go to "Version Control"
3. Click "Connect to GitHub"
4. Authorize Replit access
5. Select repository: `valpere/shopogoda`
6. Enable "Run on every commit"

**Now:**
- Push to GitHub → Repl auto-deploys
- Automatic restarts on code changes

### Environment-Specific Config

Use different configs for dev/staging/prod:

**Add ENV secret:**
```
Key: ENV
Value: production
```

**Load config based on ENV:**
```go
env := os.Getenv("ENV")
if env == "" {
    env = "development"
}

configFile := fmt.Sprintf("config.%s.yaml", env)
```

**Create config files:**
- `config.development.yaml`
- `config.staging.yaml`
- `config.production.yaml`

### Scheduled Tasks

For periodic tasks (weather updates, cleanup):

**Option 1: Use internal scheduler** (existing)
```go
// Already implemented in internal/services/scheduler_service.go
// Runs every hour
```

**Option 2: Use external cron**
- Set up cron on external server
- Call HTTP endpoint: `https://your-repl.repl.co/api/cron`
- Secure with API key

## Migration Guide

### From Local to Replit

1. Push code to GitHub
2. Import from GitHub to Replit
3. Set up secrets (copy from local .env)
4. Enable Replit database
5. Run and test

### From Replit to Another Platform

**To Vercel:**
1. Push code to GitHub
2. Connect Vercel to GitHub
3. Set environment variables in Vercel
4. Change to webhook mode
5. Deploy

**To Fly.io:**
1. Install Fly CLI
2. Use existing `fly.toml` config
3. Set secrets: `fly secrets set KEY=value`
4. Deploy: `fly deploy`

**To VPS:**
1. Use Docker Compose from repo
2. Copy environment variables
3. Deploy with: `docker-compose up -d`

## Getting Help

### Replit Support

**Community Forum:**
- https://ask.replit.com
- Search existing questions
- Post new questions with [Go] tag

**Discord:**
- Join Replit Discord: https://replit.com/discord
- #help channel for support

**Documentation:**
- https://docs.replit.com
- Guides for Go, databases, deployments

### Bot-Specific Help

**Project Documentation:**
- See `docs/` folder in repository
- `TROUBLESHOOTING.md` for common issues
- `TESTING.md` for testing guide

**GitHub Issues:**
- Report bugs: https://github.com/valpere/shopogoda/issues
- Feature requests welcome

**Telegram Bot API:**
- Official docs: https://core.telegram.org/bots/api
- Bot FAQ: https://core.telegram.org/bots/faq

## Conclusion

You now have a production-ready Telegram bot running on Replit!

### What You've Accomplished

✅ Deployed bot to Replit cloud
✅ Configured PostgreSQL database
✅ Set up environment variables
✅ Tested bot on Telegram
✅ (Optional) Enabled always-on mode

### Next Steps

1. **Monitor your bot** for first 24 hours
2. **Test all commands** thoroughly
3. **Set up monitoring** (logs, metrics)
4. **Consider upgrading** to Pro for 24/7 uptime
5. **Invite users** to test your bot

### Production Checklist

Before going live:

- [ ] Bot responds to all commands
- [ ] Database migrations completed
- [ ] Environment variables verified
- [ ] Always-On enabled (Pro plan)
- [ ] Error handling tested
- [ ] Monitoring set up
- [ ] Backup strategy in place
- [ ] Rate limiting working
- [ ] User feedback collected

### Resources

- **Replit Dashboard**: https://replit.com/~
- **Bot on Telegram**: https://t.me/your_bot_username
- **Project Repository**: https://github.com/valpere/shopogoda
- **OpenWeather API**: https://openweathermap.org/api
- **Telegram Bot API**: https://core.telegram.org/bots/api

---

**Document Version**: 1.0
**Last Updated**: 2025-10-06
**Platform**: Replit
**Tested With**: Go 1.21, PostgreSQL 14, gotgbot v2.0.0-rc.25
**Author**: ShoPogoda Team
