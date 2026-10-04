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
3. Docker Compose: `docker compose logs -f bot`; systemd: `journalctl -u shopogoda -f`; Kubernetes: `kubectl -n shopogoda logs -f deployment/shopogoda`

### Method 3: From Database
Query the `users` table with `sqlite3` to see all registered users and their IDs (`SELECT id, username FROM users;`).

## Granting Admin in the SQLite Database

ShoPogoda stores everything in one SQLite file (`DB_PATH`, default
`./data/shopogoda.db`; `/app/data/shopogoda.db` in the container). The user must
have sent `/start` to the bot first so that their row exists in `users`.

Roles are stored as integers in `users.role`: 1 = User, 2 = Moderator, 3 = Admin.

### Option 1: Bare binary / local development

```bash
sqlite3 ./data/shopogoda.db "UPDATE users SET role = 3 WHERE id = YOUR_TELEGRAM_USER_ID;"

# Verify
sqlite3 -header -column ./data/shopogoda.db \
  "SELECT id, username, first_name, last_name, role FROM users WHERE id = YOUR_TELEGRAM_USER_ID;"
```

Use the path from your `DB_PATH` (for a systemd install, for example
`/var/lib/shopogoda/data/shopogoda.db`, as the `shopogoda` user or with `sudo`).

### Option 2: Docker (named volume)

The application image does not include the `sqlite3` CLI, so run it from a
throwaway container against the volume. Stop the bot first to avoid lock
contention (a brief write is fine in WAL mode, but stopping is the safe choice):

```bash
docker compose -f docker/docker-compose.prod.yml stop bot

docker run --rm -v shopogoda_data_prod:/data keinos/sqlite3 \
  sqlite3 /data/shopogoda.db "UPDATE users SET role = 3 WHERE id = YOUR_TELEGRAM_USER_ID;"

docker compose -f docker/docker-compose.prod.yml start bot
```

Volume names: `shopogoda_data_prod` (production compose), `shopogoda_data_staging`
(staging compose), or whatever you passed to `docker run -v`. Alternatively,
install `sqlite3` on the host and edit the file at the path shown by
`docker volume inspect <volume>` (stop the bot first and run as root).

### Option 3: Kubernetes

Scale the deployment to zero, run a one-off pod that mounts the
`shopogoda-data` PVC and executes the same `sqlite3` UPDATE (for example with the
`keinos/sqlite3` image), then scale back to one replica. Never run two pods
against the same database file.

### Option 4: `/promote` command

Once at least one admin exists, further roles are managed from Telegram (see
below). The first admin must be granted directly in the database.

## Verification

After granting admin access, verify it works:

1. **Clear bot cache (if needed):**
   - User profiles are cached in memory with a 1-hour TTL
   - Wait up to 1 hour or restart the bot to clear the cache immediately

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

User profiles are cached in memory for 1 hour (and invalidated when the bot
itself updates the user). A manual database edit is therefore picked up after the
cache TTL, or immediately after restarting the bot:

```bash
# Docker Compose
docker compose -f docker/docker-compose.prod.yml restart bot

# systemd
sudo systemctl restart shopogoda

# Kubernetes
kubectl -n shopogoda rollout restart deployment/shopogoda
```

### Issue: "unable to open database file" / "database is locked"

- Check that you pointed `sqlite3` at the right file (`DB_PATH`).
- "locked" means another process holds a write lock; stop the bot or close other
  `sqlite3` shells and retry.
- If you edited the file as root, make sure ownership still matches the bot user.

### Issue: "User ID not found in database"

1. Interact with the bot first (send `/start`)
2. Users are created on first interaction
3. Check the users table:
   ```bash
   sqlite3 -header -column ./data/shopogoda.db \
     "SELECT id, username, first_name, role, created_at FROM users ORDER BY created_at DESC LIMIT 10;"
   ```

## Security Considerations

1. **Limit Admin Access:** Only grant admin role to trusted individuals
2. **Use Moderator Role:** For most moderation tasks, Moderator role is sufficient
3. **Audit Logs:** All role changes are logged with admin ID and timestamp
4. **Database Access:** Restrict direct database access to authorized personnel
5. **Backups:** The SQLite file and its backups contain user data; keep them readable only by the bot user

## Reference

- **User Service:** `internal/services/user_service.go` (ChangeUserRole method)
- **Admin Commands:** `internal/handlers/commands/admin.go`
- **User Model:** `internal/models/models.go` (User struct, UserRole constants)
- **Bot Commands:** See `/help` in the bot for full command reference

## Related Documentation

- [Configuration Guide](CONFIGURATION.md) - Environment setup
- [Deployment Guide](DEPLOYMENT.md) - Production deployment
- [API Reference](API_REFERENCE.md) - Service layer documentation
