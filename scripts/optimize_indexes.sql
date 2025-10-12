-- Optimize Database Indexes for ShoPogoda
-- This script replaces individual indexes with composite indexes for better query performance
-- Run this migration in Supabase SQL Editor: https://supabase.com/dashboard/project/_/sql

-- ====================================================================================
-- OVERVIEW
-- ====================================================================================
-- Supabase Database Linter reported 6 "unused" indexes. Analysis shows these indexes
-- are actually used in production queries, but can be optimized by combining them into
-- composite indexes.
--
-- Benefits of composite indexes:
-- 1. Better query performance (single index scan vs multiple lookups)
-- 2. Reduced storage overhead (fewer indexes to maintain)
-- 3. Lower maintenance cost (fewer indexes to update on writes)
-- 4. Improved cache utilization
--
-- This migration:
-- 1. Creates composite indexes for common query patterns
-- 2. Drops redundant individual indexes
-- 3. Keeps username index (future feature support)
-- 4. Optimizes for actual production query patterns
-- ====================================================================================

BEGIN;

-- ====================================================================================
-- 1. WEATHER_DATA TABLE OPTIMIZATION
-- ====================================================================================
-- Current queries use: WHERE user_id = ? AND timestamp >= ? ORDER BY timestamp DESC
-- Current indexes: idx_weather_data_user_id, idx_weather_data_timestamp (separate)
-- New approach: Single composite index covering both columns

-- Query pattern from export_service.go:165-166:
--   Where("user_id = ? AND timestamp >= ?", userID, thirtyDaysAgo).
--   Order("timestamp DESC").

-- Drop individual indexes
DROP INDEX IF EXISTS public.idx_weather_data_user_id;
DROP INDEX IF EXISTS public.idx_weather_data_timestamp;

-- Create optimized composite index
-- Column order: user_id first (equality), then timestamp (range + sort)
-- DESC on timestamp matches ORDER BY clause for optimal performance
CREATE INDEX idx_weather_data_user_timestamp
ON public.weather_data(user_id, timestamp DESC);

COMMENT ON INDEX public.idx_weather_data_user_timestamp IS
'Composite index for weather data export queries. Covers user_id filtering and timestamp sorting in a single index scan.';

-- ====================================================================================
-- 2. ENVIRONMENTAL_ALERTS TABLE OPTIMIZATION
-- ====================================================================================
-- Current queries use: WHERE user_id = ? AND created_at >= ? ORDER BY created_at DESC
-- Current indexes: idx_environmental_alerts_user_id, idx_environmental_alerts_created_at (separate)
-- New approach: Single composite index covering both columns

-- Query pattern from export_service.go:187-189:
--   Where("user_id = ? AND created_at >= ?", userID, ninetyDaysAgo).
--   Order("created_at DESC").

-- Drop individual indexes
DROP INDEX IF EXISTS public.idx_environmental_alerts_user_id;
DROP INDEX IF EXISTS public.idx_environmental_alerts_created_at;

-- Create optimized composite index
-- Column order: user_id first (equality), then created_at (range + sort)
-- DESC on created_at matches ORDER BY clause
CREATE INDEX idx_environmental_alerts_user_created
ON public.environmental_alerts(user_id, created_at DESC);

COMMENT ON INDEX public.idx_environmental_alerts_user_created IS
'Composite index for alert history queries. Covers user_id filtering and created_at sorting in a single index scan.';

-- ====================================================================================
-- 3. SUBSCRIPTIONS TABLE
-- ====================================================================================
-- Current queries use: WHERE user_id = ?
-- Current index: idx_subscriptions_user_id
-- Action: KEEP AS-IS (already optimal for single-column queries)

-- Query patterns:
--   subscription_service.go: Where("user_id = ?", userID)
--   export_service.go: Where("user_id = ?", userID)

-- No optimization needed - single column index is already optimal
COMMENT ON INDEX public.idx_subscriptions_user_id IS
'Optimized for user subscription lookups. No composite index needed for single-column queries.';

-- ====================================================================================
-- 4. USERS TABLE
-- ====================================================================================
-- Current index: idx_users_username
-- Action: KEEP AS-IS (needed for future features and admin panel)

-- Reasons to keep:
-- 1. Standard best practice for user lookup
-- 2. Used by Supabase admin dashboard
-- 3. Will be needed for future features (user search, admin dashboard)
-- 4. Minimal storage overhead

COMMENT ON INDEX public.idx_users_username IS
'Username lookup index. Reserved for future features (user search, admin dashboard). Minimal overhead.';

-- ====================================================================================
-- VERIFICATION
-- ====================================================================================
-- After running this migration, verify indexes are created:
--
-- SELECT
--   schemaname,
--   tablename,
--   indexname,
--   indexdef
-- FROM pg_indexes
-- WHERE schemaname = 'public'
-- AND tablename IN ('users', 'weather_data', 'subscriptions', 'environmental_alerts')
-- ORDER BY tablename, indexname;
--
-- Expected results:
-- - weather_data: idx_weather_data_user_timestamp (composite)
-- - environmental_alerts: idx_environmental_alerts_user_created (composite)
-- - subscriptions: idx_subscriptions_user_id (single column - kept)
-- - users: idx_users_username (single column - kept)
--
-- Check index usage after some queries:
--
-- SELECT
--   schemaname,
--   tablename,
--   indexname,
--   idx_scan,
--   idx_tup_read,
--   idx_tup_fetch
-- FROM pg_stat_user_indexes
-- WHERE schemaname = 'public'
-- ORDER BY tablename, indexname;
--
-- After running production queries, you should see:
-- - idx_weather_data_user_timestamp: idx_scan > 0
-- - idx_environmental_alerts_user_created: idx_scan > 0
-- ====================================================================================

COMMIT;

-- ====================================================================================
-- PERFORMANCE ANALYSIS
-- ====================================================================================
-- Before optimization (6 indexes):
-- - weather_data: 2 indexes (user_id, timestamp)
-- - environmental_alerts: 2 indexes (user_id, created_at)
-- - subscriptions: 1 index (user_id)
-- - users: 1 index (username)
-- Total: 6 indexes
--
-- After optimization (4 indexes):
-- - weather_data: 1 composite index (user_id, timestamp DESC)
-- - environmental_alerts: 1 composite index (user_id, created_at DESC)
-- - subscriptions: 1 index (user_id)
-- - users: 1 index (username)
-- Total: 4 indexes
--
-- Benefits:
-- - 33% reduction in index count (6 → 4)
-- - Faster queries (single index scan vs multiple lookups)
-- - Lower storage overhead
-- - Reduced write amplification (fewer indexes to update)
-- - Better cache hit rates
-- ====================================================================================

-- ====================================================================================
-- QUERY PERFORMANCE EXPECTATIONS
-- ====================================================================================
-- 1. Weather Data Export Query:
--    Before: Sequential scan on user_id index, then filter/sort on timestamp
--    After:  Single index scan on composite index (user_id, timestamp DESC)
--    Expected improvement: 2-3x faster on large datasets
--
-- 2. Alert History Query:
--    Before: Sequential scan on user_id index, then filter/sort on created_at
--    After:  Single index scan on composite index (user_id, created_at DESC)
--    Expected improvement: 2-3x faster on large datasets
--
-- 3. Subscription Queries:
--    No change - already optimal
--
-- 4. User Queries:
--    No change - index reserved for future use
-- ====================================================================================

-- ====================================================================================
-- ROLLBACK SCRIPT (if needed)
-- ====================================================================================
-- To restore original indexes:
--
-- BEGIN;
--
-- -- Restore weather_data indexes
-- DROP INDEX IF EXISTS public.idx_weather_data_user_timestamp;
-- CREATE INDEX idx_weather_data_user_id ON public.weather_data(user_id);
-- CREATE INDEX idx_weather_data_timestamp ON public.weather_data(timestamp);
--
-- -- Restore environmental_alerts indexes
-- DROP INDEX IF EXISTS public.idx_environmental_alerts_user_created;
-- CREATE INDEX idx_environmental_alerts_user_id ON public.environmental_alerts(user_id);
-- CREATE INDEX idx_environmental_alerts_created_at ON public.environmental_alerts(created_at);
--
-- COMMIT;
-- ====================================================================================

-- ====================================================================================
-- NOTES FOR MONITORING
-- ====================================================================================
-- Monitor index usage over the next 7 days:
--
-- 1. Check index scans:
--    SELECT * FROM pg_stat_user_indexes
--    WHERE schemaname = 'public'
--    ORDER BY idx_scan DESC;
--
-- 2. Identify slow queries:
--    Check Supabase Dashboard → Performance → Slow Queries
--
-- 3. Analyze query plans:
--    EXPLAIN ANALYZE SELECT * FROM weather_data
--    WHERE user_id = 123 AND timestamp >= NOW() - INTERVAL '30 days'
--    ORDER BY timestamp DESC;
--
-- Expected query plan should show:
-- "Index Scan using idx_weather_data_user_timestamp"
--
-- If you see "Seq Scan" or "Bitmap Heap Scan", the index may need adjustment.
-- ====================================================================================

-- ====================================================================================
-- FUTURE OPTIMIZATION OPPORTUNITIES
-- ====================================================================================
-- If alert_configs table grows large, consider:
--
-- CREATE INDEX idx_alert_configs_user_active
-- ON public.alert_configs(user_id, is_active)
-- WHERE is_active = true;
--
-- This partial index would optimize:
-- - Active alert lookups (scheduler_service.go)
-- - Export of active alert configs
-- - Smaller index size (only active alerts)
-- ====================================================================================
