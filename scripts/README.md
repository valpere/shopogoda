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

## Related Documentation

- [Database Security Guide](../docs/DATABASE_SECURITY.md)
- [Deployment Guide](../docs/DEPLOYMENT_RAILWAY.md)
- [Testing Guide](../docs/TESTING.md)
