# Scripts

Helper scripts for ShoPogoda. The database is a single SQLite file created and
migrated automatically by the bot on startup, so there are no SQL setup scripts.

| Script | Purpose |
|--------|---------|
| `migrate.go` | Run GORM AutoMigrate standalone (creates the SQLite file if missing) |
| `deploy/deploy.sh` | Deploy staging/production with docker compose |
| `deploy/health-check.sh` | Health check of a compose deployment |
| `deploy/rollback.sh` | Restore the saved env file and restart a compose deployment |
| `create-release.sh` | Update CHANGELOG, commit and create a release git tag |
| `rollback.sh` | Release rollback helper (confirmation flow only; see note) |
| `convert_locales.pl` | Convert locale files between CSV and JSON |
| `filter_locales.pl` | Remove unused keys from locale files |

## Database

### `migrate.go`

Loads the configuration, opens the SQLite database at `DB_PATH` (default
`./data/shopogoda.db`) and runs `models.Migrate` (GORM AutoMigrate).

```bash
make migrate                      # same as: go run scripts/migrate.go
DB_PATH=/tmp/test.db make migrate # use another file
make db-reset                     # DESTROYS the database file (and -wal/-shm), then migrates
```

The bot also migrates on startup, so running this by hand is optional.

### Backups

Not a script; see [DEPLOYMENT.md](../docs/DEPLOYMENT.md#backups-and-restore):

```bash
sqlite3 ./data/shopogoda.db ".backup backup.db"   # safe while the bot runs (WAL)
```

## Deployment (`deploy/`)

These scripts drive `docker/docker-compose.<environment>.yml` using the
`docker-compose` command and expect `.env.staging` / `.env.production` in the
project root (create them from `.env.staging.example` / `.env.prod.example`).

```bash
./scripts/deploy/deploy.sh staging develop
./scripts/deploy/deploy.sh production v1.2.3
./scripts/deploy/health-check.sh production
./scripts/deploy/rollback.sh production
```

- **`deploy.sh [staging|production] [version]`** - checks docker and the env
  file, saves a snapshot (container list, image list, copy of the env file) to
  `backups/<timestamp>/`, pulls images, runs `up -d`, and waits for the
  containers to become healthy.
- **`health-check.sh [staging|production]`** - checks container state and
  health, HTTP endpoints, that the database file exists in the bot container,
  and resource usage.
- **`rollback.sh [staging|production]`** - lists saved snapshots, restores the
  env file from the chosen one and restarts the stack.

The snapshots do **not** include the SQLite database. Take a database backup
before upgrading. Because SQLite allows a single writer, a deployment restarts
the single bot instance (brief downtime).

## Releases

### `create-release.sh [version]`

Validates the version (e.g. `v0.1.0-demo`), requires a clean git tree, updates
`CHANGELOG.md`, commits it, and creates an annotated tag. Pushing the tag
triggers the release workflow. See [RELEASE_PROCESS.md](../docs/RELEASE_PROCESS.md).

### `rollback.sh [version] [environment]`

Lists recent tags and asks for confirmation, but the actual rollback commands
(`kubectl rollout undo`, `helm rollback`) are commented-out placeholders: it
does not change a deployment by itself. Use `deploy/rollback.sh` (compose) or
redeploy the previous image tag, see [DEPLOYMENT.md](../docs/DEPLOYMENT.md#upgrades-and-rollback).

## Localization (Perl)

Require Perl 5.38+ with `JSON` and `Text::CSV`. Locale CSVs live in `locales/`,
JSON files in `internal/locales/`.

```bash
perl scripts/convert_locales.pl                  # CSV -> JSON (default)
perl scripts/convert_locales.pl --json-to-csv    # JSON -> CSV
perl scripts/filter_locales.pl                   # filter CSV to keys in locales/keys-cod.csv
perl scripts/filter_locales.pl --json            # filter JSON files
```

Run `perldoc scripts/<name>.pl` for all options (`--verbose`, `--keys`,
`--add_empty`, ...).
