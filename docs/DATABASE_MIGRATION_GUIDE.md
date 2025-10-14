# Database Migration Guide

Complete guide for database setup and migrations for ShoPogoda deployments.

## Quick Answer

**Do you need to run SQL patches?**

| Deployment Type | SQL Patches Required? | Reason |
|----------------|----------------------|--------|
| **New deployment (empty database)** | ❌ No | GORM AutoMigrate creates all tables automatically |
| **Existing deployment (upgrading)** | ✅ Yes | Manual SQL patches for security/optimization |
| **Local development** | ❌ No | AutoMigrate handles everything |
| **Production (Supabase with pooler)** | ⚠️ Maybe | AutoMigrate disabled, tables created on first run |

## How Database Schema Creation Works

### Automatic Schema Creation (GORM AutoMigrate)

ShoPogoda uses GORM's AutoMigrate to automatically create and update database schema:

**Location**: `internal/database/database.go:72-82`

```go
// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
 return db.AutoMigrate(
  &models.User{},
  &models.WeatherData{},
  &models.Subscription{},
  &models.AlertConfig{},
  &models.EnvironmentalAlert{},
  &models.UserSession{},
 )
}
```

**When It Runs**: Automatically on bot startup (called from `internal/bot/bot.go`)

**What It Does**:

1. Connects to database
2. Checks if tables exist
3. Creates missing tables with correct schema
4. Creates indexes defined in model tags (`gorm:"index"`)
5. Updates column types if models changed

**What It Does NOT Do**:

- ❌ Create Row Level Security (RLS) policies
- ❌ Create composite indexes (only single-column indexes from tags)
- ❌ Optimize existing indexes
- ❌ Create comments on tables/indexes

### When AutoMigrate Is Disabled

**Supabase Connection Pooler (Transaction Mode)**:

- AutoMigrate uses prepared statements for schema inspection
- Supabase transaction pooler doesn't support prepared statements
- **Workaround**: `PreferSimpleProtocol: true` in GORM config (line 24)
- **Result**: Tables are still created automatically on first deployment

**Railway Production**:

- AutoMigrate is enabled and works correctly
- Uses direct database connection (not pooler)
- All tables and indexes created automatically

## SQL Patches Overview

ShoPogoda includes 2 SQL migration scripts for **security and performance optimization**:

### 1. `scripts/enable_rls.sql` (Security - PR #88)

**Purpose**: Enable Row Level Security on Supabase tables to block PostgREST API access.

**Required?** ✅ Yes, for **Supabase deployments** (security best practice)

**When to Run**:

- After initial Supabase deployment
- When Supabase Database Linter shows "RLS Disabled" warnings
- For production security hardening

**What It Does**:

- Enables RLS on 6 tables: `users`, `weather_data`, `subscriptions`, `alert_configs`, `environmental_alerts`, `user_sessions`
- Creates deny-by-default policies for PostgREST API (`anon` and `authenticated` roles)
- Bot continues working via service role (bypasses RLS)

**Impact if NOT Run**:

- ⚠️ **Security Risk**: PostgREST API exposes all tables publicly
- Anyone with API URL could potentially read/write data
- Supabase shows ERROR-level warnings in Database Linter

**Does NOT Affect**:

- ✅ Bot functionality (bot uses service role)
- ✅ Table creation (handled by AutoMigrate)
- ✅ Data integrity

### 2. `scripts/optimize_indexes.sql` (Performance - PR #89)

**Purpose**: Optimize database indexes by replacing individual indexes with composite indexes.

**Required?** ❌ No, **optional performance optimization**

**When to Run**:

- After initial deployment (any platform)
- When Supabase Database Linter shows "unused index" warnings
- When query performance is slow for exports/history

**What It Does**:

- Replaces 2 single-column indexes on `weather_data` with 1 composite index: `(user_id, timestamp DESC)`
- Replaces 2 single-column indexes on `environmental_alerts` with 1 composite index: `(user_id, created_at DESC)`
- Keeps `subscriptions` and `users` indexes unchanged
- Reduces index count by 33% (6 → 4 indexes)

**Impact if NOT Run**:

- ⚠️ **Slower Queries**: Weather data export and alert history 2-3x slower
- ⚠️ **Higher Storage**: More indexes to maintain
- ⚠️ **Increased Write Cost**: More indexes to update on inserts
- ✅ **Bot Still Works**: Just with lower performance

**Does NOT Affect**:

- ✅ Bot functionality (all queries still work)
- ✅ Data integrity

## Deployment Scenarios

### Scenario 1: New Local Development Setup

**Setup**:

```bash
make init    # Start PostgreSQL + Redis containers
make run     # Start bot
```

**SQL Patches Needed**: ❌ None

**What Happens**:

1. PostgreSQL container starts with empty database
2. Bot connects to database
3. GORM AutoMigrate creates all tables automatically
4. Single-column indexes created from model tags
5. Bot starts successfully

**Result**: Ready to develop! No manual SQL needed.

### Scenario 2: New Railway Deployment (Integrated)

**Setup**:

1. Deploy to Railway from GitHub
2. Add PostgreSQL and Redis services
3. Set environment variables
4. Railway auto-deploys

**SQL Patches Needed**: ❌ None (✅ Optional: `optimize_indexes.sql`)

**What Happens**:

1. Railway creates PostgreSQL instance
2. Bot deploys and connects to database
3. GORM AutoMigrate creates all tables
4. Single-column indexes created automatically
5. Bot starts successfully

**Optimization (Optional)**:

- After deployment, run `optimize_indexes.sql` for 2-3x faster queries
- Not required for bot to work

**Result**: Production-ready! SQL patches optional for optimization.

### Scenario 3: New Supabase Deployment (Hybrid)

**Setup**:

1. Create Supabase project
2. Deploy bot to Railway (or Vercel/Fly.io)
3. Configure Supabase connection string (pooler port 6543)
4. Set environment variables
5. Bot deploys

**SQL Patches Needed**:

- ✅ **Required**: `enable_rls.sql` (security)
- ✅ **Recommended**: `optimize_indexes.sql` (performance)

**What Happens**:

1. Supabase creates PostgreSQL database
2. Bot connects via connection pooler (port 6543)
3. GORM AutoMigrate creates all tables (with `PreferSimpleProtocol: true`)
4. Single-column indexes created automatically
5. ⚠️ **Supabase exposes tables via PostgREST API** (insecure by default!)

**Security Fix (REQUIRED)**:

```bash
# Run enable_rls.sql in Supabase SQL Editor
# Blocks PostgREST API, bot continues working
```

**Optimization (Recommended)**:

```bash
# Run optimize_indexes.sql for better query performance
# 2-3x faster weather data export and alert history
```

**Result**:

- Without RLS: ⚠️ Functional but **INSECURE** (PostgREST API exposed)
- With RLS: ✅ Secure and functional
- With RLS + optimization: ✅ Secure, functional, and **FAST**

### Scenario 4: Existing Production Deployment (Upgrading)

**Scenario**: You deployed ShoPogoda months ago, now upgrading to latest version.

**SQL Patches Needed**: ✅ Yes, both scripts

**What Happens**:

1. Pull latest code from GitHub
2. Redeploy to Railway/Vercel/Fly.io
3. GORM AutoMigrate detects existing schema
4. **AutoMigrate updates changed columns only**
5. ❌ **Does NOT add RLS policies**
6. ❌ **Does NOT optimize indexes**

**Required Actions**:

1. **Run `enable_rls.sql`** (if on Supabase)
   - One-time security hardening
   - Prevents data exposure via PostgREST API
2. **Run `optimize_indexes.sql`** (any platform)
   - One-time performance optimization
   - Improves query speed for exports/history

**Result**: Upgraded with security and performance improvements.

### Scenario 5: Supabase with Direct Connection (Not Pooler)

**Setup**:

```bash
# Using port 5432 (direct) instead of 6543 (pooler)
DB_HOST=db.xxxxx.supabase.co
DB_PORT=5432
```

**SQL Patches Needed**: ✅ `enable_rls.sql` (security)

**What Happens**:

1. Bot connects directly to PostgreSQL (not pooler)
2. GORM AutoMigrate works normally with prepared statements
3. All tables and indexes created automatically
4. ⚠️ **PostgREST API still exposed** (security risk)

**Required Actions**:

1. Run `enable_rls.sql` to secure PostgREST API

**Result**: Secure and functional.

## How to Run SQL Patches

### Method 1: Supabase SQL Editor (Recommended)

**For**: `enable_rls.sql` and `optimize_indexes.sql` on Supabase

**Steps**:

1. Open [Supabase Dashboard](https://supabase.com/dashboard)
2. Navigate to your project
3. Go to **SQL Editor**
4. Copy entire contents of script
5. Paste into editor
6. Click **"Run"**
7. Verify success (see verification sections below)

### Method 2: psql Command Line

**For**: Any PostgreSQL database (Supabase, Railway, local)

**Steps**:

```bash
# Connect to database
psql "postgresql://user:password@host:port/database?sslmode=require"

# Run script
\i scripts/enable_rls.sql
\i scripts/optimize_indexes.sql

# Or pipe script
psql "connection_string" < scripts/enable_rls.sql
```

### Method 3: Railway CLI

**For**: Railway PostgreSQL service

**Steps**:

```bash
# Connect to Railway database
railway run --service postgres psql

# Inside psql:
\i scripts/optimize_indexes.sql
```

## Verification

### Verify RLS is Enabled

```sql
-- Check RLS is enabled on all tables
SELECT schemaname, tablename, rowsecurity
FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN ('users', 'weather_data', 'subscriptions',
                  'alert_configs', 'environmental_alerts', 'user_sessions')
ORDER BY tablename;
```

**Expected Output**: All tables should have `rowsecurity = true`

### Verify RLS Policies Exist

```sql
-- Check policies are created
SELECT tablename, policyname, permissive, roles, cmd
FROM pg_policies
WHERE schemaname = 'public'
ORDER BY tablename, policyname;
```

**Expected Output**: 6 policies (one per table) for `anon` and `authenticated` roles

### Verify Composite Indexes

```sql
-- Check composite indexes exist
SELECT schemaname, tablename, indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
AND indexname LIKE '%_user_%'
ORDER BY tablename, indexname;
```

**Expected Output**:

- `idx_weather_data_user_timestamp` on `weather_data`
- `idx_environmental_alerts_user_created` on `environmental_alerts`

### Verify Bot Functionality

```bash
# Health check
curl https://your-app.up.railway.app/health

# Expected: {"status":"healthy","timestamp":"..."}

# Test bot commands
# Telegram → Your bot
/start      → Welcome message
/weather    → Current weather
/setlocation → Set location (tests database write)
```

## Troubleshooting

### Issue: Bot fails to start after running SQL patches

**Symptom**: Bot crashes on startup with database errors

**Cause**: Using wrong database credentials (not service role)

**Solution**:

```bash
# For Supabase: Ensure using service role credentials
# Supabase Dashboard → Settings → API → service_role key
# NOT the anon key!

# For Railway: Ensure using correct PostgreSQL variables
DATABASE_URL=${{Postgres.DATABASE_URL}}
DB_PASSWORD=${{Postgres.PGPASSWORD}}
```

### Issue: "insufficient arguments" error during migration

**Symptom**: AutoMigrate fails with "insufficient arguments"

**Cause**: Using Supabase connection pooler without `PreferSimpleProtocol`

**Solution**: Already fixed in latest version (line 24 of `database.go`)

```go
PreferSimpleProtocol: true, // Disables prepared statements
```

### Issue: Supabase Linter still shows RLS warnings

**Symptom**: Warnings persist after running `enable_rls.sql`

**Cause**: Supabase dashboard cache

**Solution**:

1. Re-run verification queries
2. Hard refresh dashboard (Ctrl+Shift+R)
3. Wait 1-2 minutes for cache to clear

### Issue: Queries are slow after optimization

**Symptom**: Unexpectedly slow queries after running `optimize_indexes.sql`

**Cause**: PostgreSQL needs to analyze new indexes

**Solution**:

```sql
-- Analyze tables to update query planner statistics
ANALYZE weather_data;
ANALYZE environmental_alerts;

-- Force rebuild of index statistics
REINDEX TABLE weather_data;
REINDEX TABLE environmental_alerts;
```

## Best Practices

### 1. Run SQL Patches AFTER First Deployment

**Why**: Let AutoMigrate create base schema first

**Process**:

1. Deploy bot → AutoMigrate creates tables
2. Verify bot works (send `/start` command)
3. Run `enable_rls.sql` (Supabase only)
4. Run `optimize_indexes.sql` (optional, any platform)
5. Verify bot still works

### 2. Backup Before Running SQL Patches

**Why**: Safety net in case something goes wrong

**Process**:

```bash
# Supabase: Use built-in backup
# Dashboard → Database → Backups

# Railway: Use pg_dump
railway run --service postgres pg_dump > backup.sql

# Local: Use pg_dump
pg_dump "connection_string" > backup.sql
```

### 3. Test in Development First

**Why**: Catch issues before production

**Process**:

1. Run SQL patches on local development database
2. Test all bot commands thoroughly
3. Check logs for any database errors
4. Only then apply to production

### 4. Document When Patches Are Applied

**Why**: Team awareness and audit trail

**Process**:
- Add comment to commit: "Applied enable_rls.sql on 2025-10-12"
- Update internal docs with patch dates
- Record in project changelog

## Summary Table

| SQL Script | Purpose | Required? | When to Run | Impact if Skipped |
|-----------|---------|-----------|-------------|-------------------|
| **enable_rls.sql** | Security (RLS) | ✅ Supabase<br>❌ Railway/Local | After Supabase deployment | ⚠️ PostgREST API exposed (security risk) |
| **optimize_indexes.sql** | Performance | ❌ Optional | After any deployment | ⚠️ Slower queries (2-3x), higher storage |

## Key Takeaways

1. ✅ **New deployments**: GORM AutoMigrate creates all tables automatically - **no SQL patches required** to get started
2. ✅ **Supabase security**: Run `enable_rls.sql` to secure PostgREST API - **required for production security**
3. ✅ **Performance optimization**: Run `optimize_indexes.sql` for 2-3x faster queries - **optional but recommended**
4. ✅ **Existing deployments**: Run both scripts when upgrading to latest version - **improves security and performance**
5. ✅ **Local development**: No SQL patches needed - **AutoMigrate handles everything**

## Related Documentation

- [Database Security Guide](DATABASE_SECURITY.md) - Complete RLS implementation details
- [Railway Deployment Guide](DEPLOYMENT_RAILWAY.md) - Production deployment instructions
- [Scripts README](../scripts/README.md) - Detailed script documentation

## Support

For database migration issues:

- Check verification queries in this guide
- Review script comments in `scripts/` directory
- Check Supabase/Railway dashboard logs
- [Issues](https://github.com/valpere/shopogoda/issues)

---

**Last Updated**: 2025-10-12
**Version**: ShoPogoda v1.0.2+
**Maintained by**: [@valpere](https://github.com/valpere)
