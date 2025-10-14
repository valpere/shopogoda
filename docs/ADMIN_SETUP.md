# Admin Setup Guide

This guide explains how to grant admin access to the first user (bot owner) in ShoPogoda bot. After the first admin is set up, they can use `/promote` and `/demote` commands to manage other users' roles.

## Role System Overview

ShoPogoda uses a three-tier role system:

| Role | Value | Permissions |
|------|-------|-------------|
| User | 1 | Basic bot features (weather, alerts, subscriptions) |
| Moderator | 2 | User features + moderate content, view statistics |
| Admin | 3 | All features + user management, broadcast messages, role changes |

**Important:** By default, all new users are created with the `User` role (value: 1).

## Finding Your Telegram User ID

Before granting admin access, you need your Telegram user ID. Here are several methods:

### Method 1: Using @userinfobot
1. Open Telegram
2. Search for `@userinfobot`
3. Start a chat and send any message
4. The bot will reply with your user ID

### Method 2: From Bot Logs
1. Send any command to your bot (e.g., `/start`)
2. Check the bot logs - they include the user ID
3. In local development: `docker-compose logs -f bot`
4. In production (Railway): Check the deployment logs

### Method 3: From Database
Query the `users` table to see all registered users and their IDs.

## Local Development Environment

### Option 1: Using psql (Recommended)

1. **Connect to local PostgreSQL container:**
   ```bash
   # Make sure containers are running
   make docker-up

   # Find the PostgreSQL container name
   docker ps | grep postgres
   # Should show: shopogoda-db

   # Connect to PostgreSQL
   # Credentials from docker-compose.yml:
   # - User: weather_user
   # - Password: weather_pass
   # - Database: weather_bot
   docker exec -it shopogoda-db psql -U weather_user -d weather_bot
   ```

2. **Grant admin role:**
   ```sql
   -- Replace YOUR_TELEGRAM_USER_ID with your actual ID
   UPDATE users
   SET role = 3, updated_at = NOW()
   WHERE id = YOUR_TELEGRAM_USER_ID;

   -- Verify the change
   SELECT id, username, first_name, last_name, role, created_at
   FROM users
   WHERE id = YOUR_TELEGRAM_USER_ID;
   ```

   Expected output:
   ```
        id     | username | first_name | last_name | role |     created_at
   -----------+----------+------------+-----------+------+---------------------
    123456789 | johndoe  | John       | Doe       |    3 | 2025-01-14 10:30:00
   ```

3. **Exit psql:**
   ```sql
   \q
   ```

### Option 2: Using DBeaver or pgAdmin

1. **Create connection:**
   - Host: `localhost`
   - Port: `5432`
   - Database: `weather_bot`
   - Username: `weather_user`
   - Password: `weather_pass` (from `docker/docker-compose.yml`)

2. **Execute SQL:**
   ```sql
   UPDATE users
   SET role = 3, updated_at = NOW()
   WHERE id = YOUR_TELEGRAM_USER_ID;
   ```

### Option 3: Using SQL Script

1. **Create a SQL file (`scripts/grant_admin.sql`):**
   ```sql
   -- Grant admin role to user
   -- Usage: psql -U weather_user -d weather_bot -f scripts/grant_admin.sql -v user_id=YOUR_ID

   UPDATE users
   SET role = 3, updated_at = NOW()
   WHERE id = :user_id;

   -- Display confirmation
   SELECT
       id,
       username,
       first_name || ' ' || last_name AS full_name,
       CASE role
           WHEN 1 THEN 'User'
           WHEN 2 THEN 'Moderator'
           WHEN 3 THEN 'Admin'
       END AS role_name,
       role AS role_value
   FROM users
   WHERE id = :user_id;
   ```

2. **Execute the script:**
   ```bash
   docker exec -i shopogoda-db psql -U weather_user -d weather_bot \
     -v user_id=YOUR_TELEGRAM_USER_ID \
     -f /dev/stdin < scripts/grant_admin.sql
   ```

## Production Environment (Railway + Supabase)

### Option 1: Using Supabase Dashboard (Recommended)

1. **Navigate to Supabase:**
   - Go to [https://supabase.com/dashboard](https://supabase.com/dashboard)
   - Select your ShoPogoda project

2. **Open Table Editor:**
   - Click on "Table Editor" in the left sidebar
   - Select the `users` table

3. **Find your user:**
   - Use the search/filter to find your record by Telegram ID
   - Or scroll through the table

4. **Edit the role:**
   - Click on the `role` cell for your user
   - Change the value from `1` to `3`
   - Press Enter or click outside to save

5. **Verify:**
   - The `updated_at` field should automatically update
   - Send `/stats` command to your bot to verify admin access

### Option 2: Using Supabase SQL Editor

1. **Open SQL Editor:**
   - Go to Supabase Dashboard → SQL Editor
   - Click "New query"

2. **Execute SQL:**
   ```sql
   -- Replace YOUR_TELEGRAM_USER_ID with your actual ID
   UPDATE users
   SET role = 3, updated_at = NOW()
   WHERE id = YOUR_TELEGRAM_USER_ID;

   -- Verify the change
   SELECT
       id,
       username,
       first_name || ' ' || last_name AS full_name,
       CASE role
           WHEN 1 THEN 'User'
           WHEN 2 THEN 'Moderator'
           WHEN 3 THEN 'Admin'
       END AS role_name,
       role AS role_value,
       updated_at
   FROM users
   WHERE id = YOUR_TELEGRAM_USER_ID;
   ```

3. **Run the query:**
   - Click "Run" (or press Ctrl+Enter)
   - Check the results to confirm the role change

### Option 3: Using psql with Supabase

1. **Get connection string:**
   - Go to Supabase Dashboard → Project Settings → Database
   - Copy the connection string (choose "Connection pooling" for better performance)
   - Replace `[YOUR-PASSWORD]` with your database password

2. **Connect via psql:**
   ```bash
   psql "postgresql://postgres.xxxxx:[YOUR-PASSWORD]@aws-0-us-east-1.pooler.supabase.com:6543/postgres"
   ```

3. **Update role:**
   ```sql
   UPDATE users
   SET role = 3, updated_at = NOW()
   WHERE id = YOUR_TELEGRAM_USER_ID;

   \q
   ```

## Verification

After granting admin access, verify it works:

1. **Clear bot cache (if needed):**
   - User data is cached in Redis with 5-minute TTL
   - Wait 5 minutes or restart the bot to clear cache immediately

2. **Test admin commands:**
   ```
   /stats          - Should show system statistics
   /users          - Should list all users
   /broadcast      - Should allow sending messages to all users
   /promote        - Should show usage for promoting users
   /demote         - Should show usage for demoting users
   ```

3. **Check settings:**
   ```
   /settings       - Should now show "Role: Admin"
   ```

## Managing Additional Admins

Once you have admin access, you can promote other users:

### Promoting a User to Moderator

```
/promote USER_ID moderator
```

Example:
```
/promote 987654321 moderator
```

### Promoting a Moderator to Admin

```
/promote USER_ID admin
```

Example:
```
/promote 987654321 admin
```

### Important Notes

- **Role Progression:** Users must be promoted through the hierarchy: User → Moderator → Admin
- **Cannot skip levels:** You cannot promote a User directly to Admin
- **Self-protection:** Admins cannot change their own role
- **Last admin protection:** Cannot demote the last admin in the system
- **Confirmation required:** All role changes require confirmation via inline keyboard

## Troubleshooting

### Issue: "Role still shows as User after update"

**Solution:**
1. Wait 5 minutes for cache to expire, OR
2. Restart the bot to clear Redis cache immediately:
   ```bash
   # Local development
   docker-compose restart bot

   # Production (Railway)
   # Redeploy from Railway dashboard or use railway CLI:
   railway up
   ```

### Issue: "Cannot connect to local database"

**Solution:**
1. Ensure containers are running:
   ```bash
   make docker-up
   docker ps  # Should show postgres container
   ```

2. Check database logs:
   ```bash
   docker logs shopogoda-db
   # Or from the docker directory:
   cd docker && docker-compose logs postgres
   ```

### Issue: "User ID not found in database"

**Solution:**
1. Interact with the bot first (send `/start`)
2. Users are created on first interaction
3. Check the users table:
   ```sql
   SELECT id, username, first_name, role, created_at
   FROM users
   ORDER BY created_at DESC
   LIMIT 10;
   ```

### Issue: "Supabase connection fails"

**Solution:**
1. Verify connection string in Railway environment variables
2. Check Supabase project is active (not paused)
3. Verify password is correct
4. Ensure IP allowlist includes Railway IPs (or disable IP restrictions)

## Security Considerations

1. **Limit Admin Access:** Only grant admin role to trusted individuals
2. **Use Moderator Role:** For most moderation tasks, Moderator role is sufficient
3. **Audit Logs:** All role changes are logged with admin ID and timestamp
4. **Database Access:** Restrict direct database access to authorized personnel
5. **Environment Variables:** Never commit database credentials to version control

## Reference

- **User Service:** `internal/services/user_service.go` (ChangeUserRole method)
- **Admin Commands:** `internal/handlers/commands/admin.go`
- **User Model:** `internal/models/models.go` (User struct, UserRole constants)
- **Bot Commands:** See `/help` in the bot for full command reference

## Related Documentation

- [Database Security](DATABASE_SECURITY.md) - RLS policies and security
- [Configuration Guide](CONFIGURATION.md) - Environment setup
- [Deployment Guide](DEPLOYMENT_RAILWAY.md) - Production deployment
- [API Reference](API_REFERENCE.md) - Service layer documentation
