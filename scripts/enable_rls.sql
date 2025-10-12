-- Enable Row Level Security (RLS) for Supabase PostgREST API
-- This script secures all public tables by enabling RLS and creating appropriate policies
-- Run this migration in Supabase SQL Editor: https://supabase.com/dashboard/project/_/sql

-- ====================================================================================
-- OVERVIEW
-- ====================================================================================
-- Supabase automatically exposes all tables in the 'public' schema via PostgREST API.
-- By default, this API allows unrestricted access, which is a security risk.
--
-- This migration:
-- 1. Enables Row Level Security (RLS) on all tables
-- 2. Creates restrictive policies that DENY PostgREST API access by default
-- 3. Allows ONLY the authenticated database user (the bot) to access data
--
-- Result: PostgREST API will be completely blocked while the bot continues to work normally.
-- ====================================================================================

BEGIN;

-- ====================================================================================
-- 1. USERS TABLE
-- ====================================================================================
-- Contains Telegram user data with roles and locations
-- Security: Users should only access their own data via PostgREST API (blocked by default)

ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;

-- Policy: Deny all PostgREST API access to users table
-- The bot uses direct database connection with service role, so it bypasses RLS
CREATE POLICY "users_postgrest_deny_all"
ON public.users
FOR ALL
TO anon, authenticated
USING (false)
WITH CHECK (false);

-- Optional: Allow users to read only their own data (currently disabled)
-- Uncomment if you want to expose PostgREST API for user self-service
-- CREATE POLICY "users_postgrest_read_own"
-- ON public.users
-- FOR SELECT
-- TO authenticated
-- USING (auth.uid()::text = id::text);

COMMENT ON TABLE public.users IS 'Telegram users with RLS enabled. PostgREST API access denied by default. Bot uses service role connection.';

-- ====================================================================================
-- 2. WEATHER_DATA TABLE
-- ====================================================================================
-- Contains weather records linked to users
-- Security: Weather data should only be accessible by the owning user

ALTER TABLE public.weather_data ENABLE ROW LEVEL SECURITY;

-- Policy: Deny all PostgREST API access to weather_data table
CREATE POLICY "weather_data_postgrest_deny_all"
ON public.weather_data
FOR ALL
TO anon, authenticated
USING (false)
WITH CHECK (false);

-- Optional: Allow users to read their own weather data (currently disabled)
-- CREATE POLICY "weather_data_postgrest_read_own"
-- ON public.weather_data
-- FOR SELECT
-- TO authenticated
-- USING (
--   user_id::text = auth.uid()::text
-- );

COMMENT ON TABLE public.weather_data IS 'Weather records with RLS enabled. PostgREST API access denied. Bot uses service role connection.';

-- ====================================================================================
-- 3. SUBSCRIPTIONS TABLE
-- ====================================================================================
-- Contains user notification preferences
-- Security: Users should only access their own subscriptions

ALTER TABLE public.subscriptions ENABLE ROW LEVEL SECURITY;

-- Policy: Deny all PostgREST API access to subscriptions table
CREATE POLICY "subscriptions_postgrest_deny_all"
ON public.subscriptions
FOR ALL
TO anon, authenticated
USING (false)
WITH CHECK (false);

-- Optional: Allow users to manage their own subscriptions (currently disabled)
-- CREATE POLICY "subscriptions_postgrest_own"
-- ON public.subscriptions
-- FOR ALL
-- TO authenticated
-- USING (user_id::text = auth.uid()::text)
-- WITH CHECK (user_id::text = auth.uid()::text);

COMMENT ON TABLE public.subscriptions IS 'User notification preferences with RLS enabled. PostgREST API access denied. Bot uses service role connection.';

-- ====================================================================================
-- 4. ALERT_CONFIGS TABLE
-- ====================================================================================
-- Contains custom alert configurations
-- Security: Users should only access their own alert configs

ALTER TABLE public.alert_configs ENABLE ROW LEVEL SECURITY;

-- Policy: Deny all PostgREST API access to alert_configs table
CREATE POLICY "alert_configs_postgrest_deny_all"
ON public.alert_configs
FOR ALL
TO anon, authenticated
USING (false)
WITH CHECK (false);

-- Optional: Allow users to manage their own alert configs (currently disabled)
-- CREATE POLICY "alert_configs_postgrest_own"
-- ON public.alert_configs
-- FOR ALL
-- TO authenticated
-- USING (user_id::text = auth.uid()::text)
-- WITH CHECK (user_id::text = auth.uid()::text);

COMMENT ON TABLE public.alert_configs IS 'Alert configurations with RLS enabled. PostgREST API access denied. Bot uses service role connection.';

-- ====================================================================================
-- 5. ENVIRONMENTAL_ALERTS TABLE
-- ====================================================================================
-- Contains triggered alerts history
-- Security: Users should only see their own alerts

ALTER TABLE public.environmental_alerts ENABLE ROW LEVEL SECURITY;

-- Policy: Deny all PostgREST API access to environmental_alerts table
CREATE POLICY "environmental_alerts_postgrest_deny_all"
ON public.environmental_alerts
FOR ALL
TO anon, authenticated
USING (false)
WITH CHECK (false);

-- Optional: Allow users to read their own triggered alerts (currently disabled)
-- CREATE POLICY "environmental_alerts_postgrest_read_own"
-- ON public.environmental_alerts
-- FOR SELECT
-- TO authenticated
-- USING (user_id::text = auth.uid()::text);

COMMENT ON TABLE public.environmental_alerts IS 'Triggered alerts with RLS enabled. PostgREST API access denied. Bot uses service role connection.';

-- ====================================================================================
-- 6. USER_SESSIONS TABLE
-- ====================================================================================
-- Contains temporary session data
-- Security: Highly sensitive, no PostgREST API access allowed

ALTER TABLE public.user_sessions ENABLE ROW LEVEL SECURITY;

-- Policy: Deny all PostgREST API access to user_sessions table
-- This table contains sensitive session data and should NEVER be exposed via API
CREATE POLICY "user_sessions_postgrest_deny_all"
ON public.user_sessions
FOR ALL
TO anon, authenticated
USING (false)
WITH CHECK (false);

COMMENT ON TABLE public.user_sessions IS 'User sessions with RLS enabled. PostgREST API access completely denied. Bot uses service role connection.';

-- ====================================================================================
-- VERIFICATION
-- ====================================================================================
-- After running this migration, verify RLS is enabled:
--
-- SELECT schemaname, tablename, rowsecurity
-- FROM pg_tables
-- WHERE schemaname = 'public'
-- AND tablename IN ('users', 'weather_data', 'subscriptions', 'alert_configs', 'environmental_alerts', 'user_sessions')
-- ORDER BY tablename;
--
-- Expected result: All tables should have rowsecurity = true
--
-- Verify policies exist:
--
-- SELECT schemaname, tablename, policyname, permissive, roles, cmd, qual, with_check
-- FROM pg_policies
-- WHERE schemaname = 'public'
-- ORDER BY tablename, policyname;
--
-- Expected result: One 'deny_all' policy per table for anon and authenticated roles
-- ====================================================================================

COMMIT;

-- ====================================================================================
-- ROLLBACK SCRIPT (if needed)
-- ====================================================================================
-- To disable RLS and remove policies (NOT recommended for production):
--
-- BEGIN;
-- DROP POLICY IF EXISTS "users_postgrest_deny_all" ON public.users;
-- DROP POLICY IF EXISTS "weather_data_postgrest_deny_all" ON public.weather_data;
-- DROP POLICY IF EXISTS "subscriptions_postgrest_deny_all" ON public.subscriptions;
-- DROP POLICY IF EXISTS "alert_configs_postgrest_deny_all" ON public.alert_configs;
-- DROP POLICY IF EXISTS "environmental_alerts_postgrest_deny_all" ON public.environmental_alerts;
-- DROP POLICY IF EXISTS "user_sessions_postgrest_deny_all" ON public.user_sessions;
--
-- ALTER TABLE public.users DISABLE ROW LEVEL SECURITY;
-- ALTER TABLE public.weather_data DISABLE ROW LEVEL SECURITY;
-- ALTER TABLE public.subscriptions DISABLE ROW LEVEL SECURITY;
-- ALTER TABLE public.alert_configs DISABLE ROW LEVEL SECURITY;
-- ALTER TABLE public.environmental_alerts DISABLE ROW LEVEL SECURITY;
-- ALTER TABLE public.user_sessions DISABLE ROW LEVEL SECURITY;
-- COMMIT;
-- ====================================================================================

-- ====================================================================================
-- NOTES FOR FUTURE ENHANCEMENTS
-- ====================================================================================
-- If you want to enable PostgREST API access in the future:
--
-- 1. For read-only user access:
--    - Uncomment the "read_own" policies above
--    - Users can fetch their own data via authenticated PostgREST requests
--
-- 2. For full user self-service:
--    - Uncomment the "own" policies (SELECT, INSERT, UPDATE, DELETE)
--    - Implement Supabase Auth integration in the bot
--    - Users can manage their own data via web interface
--
-- 3. For admin access:
--    - Create role-based policies checking user.role = 'Admin'
--    - Example:
--      CREATE POLICY "admin_full_access" ON public.users
--      FOR ALL TO authenticated
--      USING (
--        EXISTS (
--          SELECT 1 FROM public.users
--          WHERE id::text = auth.uid()::text AND role = 3
--        )
--      );
--
-- Current approach: Maximum security - block all PostgREST API access
-- ====================================================================================
