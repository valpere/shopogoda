# Database Security - Row Level Security (RLS)

## Overview

ShoPogoda uses Supabase PostgreSQL for data storage. Supabase automatically exposes all tables in the `public` schema via PostgREST API, which creates security concerns if not properly configured.

This document explains how Row Level Security (RLS) is implemented to secure the database.

## Problem

**Before RLS Implementation:**

- Supabase exposes all `public` schema tables via PostgREST API
- PostgREST API endpoint: `https://<project-id>.supabase.co/rest/v1/`
- Without RLS, anyone with the API URL could potentially read/write data
- Supabase Database Linter shows ERROR: "RLS Disabled in Public"

**Tables at Risk:**
1. `public.users` - User profiles and location data
2. `public.weather_data` - Weather records
3. `public.subscriptions` - Notification preferences
4. `public.alert_configs` - Custom alert configurations
5. `public.environmental_alerts` - Triggered alerts history
6. `public.user_sessions` - Temporary session data (highly sensitive)

## Solution: Row Level Security (RLS)

RLS is a PostgreSQL feature that allows you to control which rows users can access in a query. Supabase uses RLS to secure PostgREST API access while allowing service role connections (like our bot) to bypass these restrictions.

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                     CLIENT ACCESS FLOW                       │
└─────────────────────────────────────────────────────────────┘

PostgREST API Request (anon/authenticated role)
        │
        ├──> RLS Policies Applied
        │    └──> Policy: DENY ALL
        │         └──> 🚫 Request Blocked
        │

Bot Database Connection (service role)
        │
        ├──> RLS Policies BYPASSED
        │    └──> Service role has superuser privileges
        │         └──> ✅ Full Access Granted
```

### Implementation Details

**1. Enable RLS on All Tables**

```sql
ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.weather_data ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.alert_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.environmental_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_sessions ENABLE ROW LEVEL SECURITY;
```

**2. Create Restrictive Policies**

We use a "deny-by-default" approach with explicit policies:

```sql
-- Example: Deny all PostgREST API access to users table
CREATE POLICY "users_postgrest_deny_all"
ON public.users
FOR ALL
TO anon, authenticated
USING (false)
WITH CHECK (false);
```

**Policy Breakdown:**
- `FOR ALL` - Applies to SELECT, INSERT, UPDATE, DELETE
- `TO anon, authenticated` - Applies to PostgREST API requests
- `USING (false)` - Read condition always fails
- `WITH CHECK (false)` - Write condition always fails

**Result:** PostgREST API is completely blocked, bot continues to work.

## Security Architecture

### Access Control Matrix

| Role | Description | RLS Applied? | Tables Access |
|------|-------------|--------------|---------------|
| `anon` | Public PostgREST API (no auth) | ✅ Yes | 🚫 Denied (all tables) |
| `authenticated` | Authenticated PostgREST API | ✅ Yes | 🚫 Denied (all tables) |
| `service_role` | Bot's database connection | ❌ No (bypasses RLS) | ✅ Full access (all tables) |

### Bot Connection

The bot uses service role credentials from environment variables:

```env
DATABASE_HOST=db.xxxxxxxxxxxxx.supabase.co
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=<service-role-password>
DATABASE_NAME=postgres
DATABASE_SSL_MODE=require
```

**Important:** The bot connects with `postgres` user (service role) which has superuser privileges and **bypasses RLS policies**.

## Deployment Instructions

> **⚠️ IMPORTANT: Manual Migration Required**
>
> This migration script **DOES NOT** run automatically. You must manually execute it in Supabase after merging the PR. The bot will NOT automatically apply these security policies - they require explicit manual deployment.
>
> **Why Manual?** Security-critical changes require intentional execution with verification steps to ensure database integrity and bot functionality are maintained.

### Option 1: Supabase SQL Editor (Recommended)

1. Log into Supabase Dashboard: https://supabase.com/dashboard
2. Navigate to your project
3. Go to "SQL Editor"
4. Copy the entire contents of `scripts/enable_rls.sql`
5. Paste into SQL Editor
6. Click "Run" to execute the migration
7. Verify success (see verification section below)

### Option 2: Supabase CLI

```bash
# Install Supabase CLI
npm install -g supabase

# Login
supabase login

# Link to your project
supabase link --project-ref <your-project-id>

# Run migration
supabase db push --dry-run  # Preview changes
supabase db push            # Apply migration
```

### Option 3: psql Command Line

```bash
# Connect to Supabase database
psql "postgresql://postgres:<password>@db.<project-id>.supabase.co:5432/postgres?sslmode=require"

# Run migration
\i scripts/enable_rls.sql
```

## Verification

After running the migration, verify RLS is properly configured:

### 1. Check RLS is Enabled

```sql
SELECT schemaname, tablename, rowsecurity
FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN ('users', 'weather_data', 'subscriptions',
                  'alert_configs', 'environmental_alerts', 'user_sessions')
ORDER BY tablename;
```

**Expected Output:**

```
 schemaname |      tablename       | rowsecurity
------------+----------------------+-------------
 public     | alert_configs        | t
 public     | environmental_alerts | t
 public     | subscriptions        | t
 public     | user_sessions        | t
 public     | users                | t
 public     | weather_data         | t
```

### 2. Check Policies Exist

```sql
SELECT tablename, policyname, permissive, roles, cmd
FROM pg_policies
WHERE schemaname = 'public'
ORDER BY tablename, policyname;
```

**Expected Output:**

```
      tablename       |          policyname           | permissive |         roles          | cmd
----------------------+-------------------------------+------------+------------------------+-----
 alert_configs        | alert_configs_postgrest_deny_all        | PERMISSIVE | {anon,authenticated}   | ALL
 environmental_alerts | environmental_alerts_postgrest_deny_all | PERMISSIVE | {anon,authenticated}   | ALL
 subscriptions        | subscriptions_postgrest_deny_all        | PERMISSIVE | {anon,authenticated}   | ALL
 user_sessions        | user_sessions_postgrest_deny_all        | PERMISSIVE | {anon,authenticated}   | ALL
 users                | users_postgrest_deny_all                | PERMISSIVE | {anon,authenticated}   | ALL
 weather_data         | weather_data_postgrest_deny_all         | PERMISSIVE | {anon,authenticated}   | ALL
```

### 3. Test Bot Functionality

After enabling RLS, the bot should continue to work normally:

```bash
# Test in production
curl https://shopogoda-svc-production.up.railway.app/health

# Test locally
make run
# Send /start command to bot
# Send /weather command to bot
# Verify all commands work correctly
```

### 4. Test PostgREST API is Blocked

Try accessing data via PostgREST API (should fail):

```bash
# Get your project details from Supabase Dashboard
PROJECT_URL="https://<your-project-id>.supabase.co"
ANON_KEY="<your-anon-key>"

# Try to read users table (should return empty or error)
curl -X GET "$PROJECT_URL/rest/v1/users" \
  -H "apikey: $ANON_KEY" \
  -H "Authorization: Bearer $ANON_KEY"

# Expected: Empty result [] or 403 Forbidden
```

## Future Enhancements

### Option 1: Read-Only User Access

If you want users to read their own data via PostgREST API:

```sql
-- Uncomment in enable_rls.sql:
CREATE POLICY "users_postgrest_read_own"
ON public.users
FOR SELECT
TO authenticated
USING (auth.uid()::text = id::text);
```

**Requirements:**
- Implement Supabase Auth in the bot
- Users must authenticate via Supabase
- Telegram user ID must match Supabase auth.uid()

### Option 2: Full User Self-Service

For a web interface where users can manage their data:

```sql
-- Allow users to manage their own subscriptions
CREATE POLICY "subscriptions_postgrest_own"
ON public.subscriptions
FOR ALL
TO authenticated
USING (user_id::text = auth.uid()::text)
WITH CHECK (user_id::text = auth.uid()::text);
```

**Benefits:**
- Users can view/edit subscriptions via web UI
- No bot interaction required for settings changes
- Enables mobile app development

### Option 3: Admin Dashboard

For admin access to all data:

```sql
CREATE POLICY "admin_full_access"
ON public.users
FOR ALL
TO authenticated
USING (
  EXISTS (
    SELECT 1 FROM public.users
    WHERE id::text = auth.uid()::text
    AND role = 3  -- RoleAdmin
  )
);
```

**Use Cases:**
- Admin web dashboard for user management
- Analytics and reporting interface
- Customer support tools

## Troubleshooting

### Bot Stopped Working After Enabling RLS

**Symptom:** Bot returns errors when trying to access database.

**Cause:** Bot is using wrong database credentials (not service role).

**Solution:**
1. Check `DATABASE_USER` is set to `postgres` (service role)
2. Check `DATABASE_PASSWORD` is the service role password (NOT the anon key)
3. Verify connection string in Railway/environment variables
4. Service role credentials are in Supabase Dashboard → Settings → API

### Supabase Linter Still Shows RLS Warnings

**Symptom:** Warnings persist after running migration.

**Cause:** Migration not applied or Supabase cache.

**Solution:**
1. Re-run verification queries
2. Refresh Supabase Dashboard (Ctrl+Shift+R)
3. Wait 1-2 minutes for cache to clear
4. Check "Table Editor" → select table → "RLS" tab shows "Enabled"

### Need to Rollback RLS

**Caution:** Only rollback if absolutely necessary. This reopens security vulnerabilities.

**Rollback Script:**

```sql
BEGIN;

-- Drop all policies
DROP POLICY IF EXISTS "users_postgrest_deny_all" ON public.users;
DROP POLICY IF EXISTS "weather_data_postgrest_deny_all" ON public.weather_data;
DROP POLICY IF EXISTS "subscriptions_postgrest_deny_all" ON public.subscriptions;
DROP POLICY IF EXISTS "alert_configs_postgrest_deny_all" ON public.alert_configs;
DROP POLICY IF EXISTS "environmental_alerts_postgrest_deny_all" ON public.environmental_alerts;
DROP POLICY IF EXISTS "user_sessions_postgrest_deny_all" ON public.user_sessions;

-- Disable RLS
ALTER TABLE public.users DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.weather_data DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.subscriptions DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.alert_configs DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.environmental_alerts DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_sessions DISABLE ROW LEVEL SECURITY;

COMMIT;
```

## Security Best Practices

### 1. Never Expose Service Role Credentials

- ❌ Don't commit service role password to git
- ❌ Don't expose in client-side code
- ❌ Don't share in public channels
- ✅ Use environment variables only
- ✅ Rotate credentials if exposed

### 2. Use Anon Key for Client Applications

- Anon key is safe to expose publicly
- RLS policies protect data even with anon key
- Rate limiting applies to anon key requests

### 3. Monitor API Access

Supabase Dashboard → Logs:
- Track PostgREST API requests
- Monitor for suspicious patterns
- Set up alerts for high request rates

### 4. Regular Security Audits

```sql
-- Review all RLS policies quarterly
SELECT * FROM pg_policies WHERE schemaname = 'public';

-- Check for tables without RLS
SELECT tablename
FROM pg_tables
WHERE schemaname = 'public'
AND rowsecurity = false;

-- List all service role users
SELECT usename, usesuper FROM pg_user;
```

## Related Documentation

- [Supabase RLS Documentation](https://supabase.com/docs/guides/database/postgres/row-level-security)
- [PostgreSQL RLS Documentation](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [Supabase Database Linter](https://supabase.com/docs/guides/database/database-linter)

## Support

For RLS-related issues:
1. Check Supabase Dashboard → Logs for error details
2. Review this documentation's troubleshooting section
3. Consult Supabase community: https://github.com/supabase/supabase/discussions
4. Contact project maintainer: valentyn.solomko@gmail.com
