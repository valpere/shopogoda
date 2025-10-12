# Database Scripts

This directory contains database management and migration scripts for ShoPogoda.

## Scripts

### `enable_rls.sql`

**Purpose:** Enable Row Level Security (RLS) on all Supabase PostgreSQL tables to secure PostgREST API access.

**When to Use:**
- After initial Supabase database setup
- When Supabase Database Linter shows RLS warnings
- For production security hardening

**How to Run:**

**Option 1: Supabase SQL Editor (Recommended)**
1. Open [Supabase Dashboard](https://supabase.com/dashboard)
2. Navigate to SQL Editor
3. Copy entire contents of `enable_rls.sql`
4. Paste and click "Run"

**Option 2: Command Line**
```bash
psql "postgresql://postgres:<password>@db.<project-id>.supabase.co:5432/postgres?sslmode=require" \
  -f scripts/enable_rls.sql
```

**What It Does:**
- Enables RLS on 6 tables: `users`, `weather_data`, `subscriptions`, `alert_configs`, `environmental_alerts`, `user_sessions`
- Creates `deny_all` policies to block PostgREST API access
- Bot continues to work normally using service role connection

**Verification:**
```sql
-- Check RLS is enabled
SELECT tablename, rowsecurity FROM pg_tables WHERE schemaname = 'public';

-- Check policies exist
SELECT tablename, policyname FROM pg_policies WHERE schemaname = 'public';
```

**Documentation:** See [docs/DATABASE_SECURITY.md](../docs/DATABASE_SECURITY.md) for complete details.

---

### `optimize_indexes.sql`

**Purpose:** Optimize database indexes by replacing individual indexes with composite indexes for better query performance.

**When to Use:**
- When Supabase Database Linter shows "unused index" warnings
- After initial database setup for performance optimization
- When query performance analysis shows index inefficiencies

**How to Run:**

**Option 1: Supabase SQL Editor (Recommended)**
1. Open [Supabase Dashboard](https://supabase.com/dashboard)
2. Navigate to SQL Editor
3. Copy entire contents of `optimize_indexes.sql`
4. Paste and click "Run"

**Option 2: Command Line**
```bash
psql "postgresql://postgres:<password>@db.<project-id>.supabase.co:5432/postgres?sslmode=require" \
  -f scripts/optimize_indexes.sql
```

**What It Does:**
- Replaces 2 individual indexes on `weather_data` with 1 composite index
- Replaces 2 individual indexes on `environmental_alerts` with 1 composite index
- Keeps `subscriptions` and `users` indexes (already optimal)
- Reduces total index count from 6 to 4 (33% reduction)
- Improves query performance by 2-3x for export and history queries

**Performance Benefits:**
- **Faster queries:** Single index scan vs multiple lookups
- **Lower storage:** Fewer indexes to store and maintain
- **Better cache utilization:** More efficient memory usage
- **Reduced write overhead:** Fewer indexes to update on inserts

**Verification:**
```sql
-- Check new composite indexes exist
SELECT schemaname, tablename, indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
AND indexname LIKE '%_user_%'
ORDER BY tablename, indexname;

-- Monitor index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;
```

**Expected Results:**
- `idx_weather_data_user_timestamp` - Composite index on (user_id, timestamp DESC)
- `idx_environmental_alerts_user_created` - Composite index on (user_id, created_at DESC)
- Old individual indexes removed
- Query performance improved

---

### `migrate.sql` (Future)

Placeholder for future database migration system.

**Planned Features:**
- Schema versioning
- Up/down migrations
- Migration history tracking
- Automated rollback support

## Migration Best Practices

1. **Always backup before migrations:**
   ```bash
   # Supabase automatic backups: Dashboard → Database → Backups
   ```

2. **Test in development first:**
   - Run migrations on local PostgreSQL
   - Verify bot functionality
   - Test edge cases

3. **Use transactions:**
   - All migration scripts use `BEGIN`/`COMMIT`
   - Automatic rollback on errors
   - Atomic changes

4. **Document changes:**
   - Update CHANGELOG.md
   - Add verification queries
   - Include rollback scripts

5. **Monitor after deployment:**
   - Check application logs
   - Verify bot commands work
   - Monitor error rates

## Rollback Procedures

If a migration causes issues, each script includes a rollback section. Always test rollback scripts before production use.

**General rollback workflow:**
1. Stop the bot application
2. Run rollback script
3. Verify database state
4. Restart bot
5. Test functionality

## Support

For database migration issues:
- Review script comments and documentation
- Check Supabase Dashboard logs
- Contact: valentyn.solomko@gmail.com

## Do I Need to Run These Scripts?

**Quick Answer**: It depends on your deployment type.

| Deployment Type | SQL Scripts Required? |
|----------------|----------------------|
| **New deployment (empty database)** | ❌ No - GORM AutoMigrate creates tables |
| **Supabase deployment** | ✅ Yes - `enable_rls.sql` (security) |
| **Railway/Local deployment** | ❌ No - Optional: `optimize_indexes.sql` (performance) |
| **Existing deployment (upgrading)** | ✅ Yes - Both scripts for security + performance |

**Detailed Explanation**: See [Database Migration Guide](../docs/DATABASE_MIGRATION_GUIDE.md)

## Related Documentation

- **[Database Migration Guide](../docs/DATABASE_MIGRATION_GUIDE.md)** - Complete guide on when to run SQL patches
- [Database Security Guide](../docs/DATABASE_SECURITY.md) - Row Level Security (RLS) implementation details
- [Deployment Guide](../docs/DEPLOYMENT_RAILWAY.md) - Production deployment instructions
- [Testing Guide](../docs/TESTING.md) - Testing documentation
