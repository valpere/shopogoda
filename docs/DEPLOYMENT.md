# Deployment Guide

This is the single deployment guide for ShoPogoda. The bot is one Go binary
(pure Go, `CGO_ENABLED=0`) with one SQLite database file. There is no external
database or cache to provision. The earlier Railway + Supabase + Upstash
production deployment has been retired.

## Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
- [Option 1: Bare Binary / systemd](#option-1-bare-binary--systemd)
- [Option 2: Docker](#option-2-docker)
- [Option 3: Docker Compose (staging / production)](#option-3-docker-compose-staging--production)
- [Option 4: Kubernetes](#option-4-kubernetes)
- [Webhook vs Polling, Reverse Proxy and HTTPS](#webhook-vs-polling-reverse-proxy-and-https)
- [Data, Volumes and Permissions](#data-volumes-and-permissions)
- [Backups and Restore](#backups-and-restore)
- [Upgrades and Rollback](#upgrades-and-rollback)
- [Health Checks and Metrics](#health-checks-and-metrics)
- [Troubleshooting](#troubleshooting)

## Overview

**Single instance only.** State lives in one SQLite file (WAL mode, single
writer). Run exactly one bot process per database file: never two replicas, and
never two containers sharing the same volume. Consequently an upgrade means a
short downtime (stop old, start new).

**In-process cache.** Weather, forecast, air-quality, geocoding and user-profile
caches live in memory and are lost on restart (they simply refill). Admin
statistics counters are also in-memory and reset on restart.

**Minimum configuration:** `TELEGRAM_BOT_TOKEN` and `OPENWEATHER_API_KEY`.
`DB_PATH` is optional (default `./data/shopogoda.db`).

| Artifact | Purpose |
|----------|---------|
| `docker/Dockerfile` | Production image (non-root user `shopogoda`, `VOLUME /app/data`) |
| `docker/docker-compose.prod.yml` | Bot + optional Prometheus, Grafana, Jaeger (production) |
| `docker/docker-compose.staging.yml` | Same, for staging |
| `deployments/k8s/*.yaml` | Namespace, ConfigMap, Secret, PVC, Deployment, Service |
| `scripts/deploy/{deploy,health-check,rollback}.sh` | Helper scripts for compose-based VM deployments |

## Configuration

All settings are environment variables; see [CONFIGURATION.md](CONFIGURATION.md)
for the full reference. Templates:

- `.env.example` - local development
- `.env.staging.example` - staging (copy to `.env.staging`)
- `.env.prod.example` - production (copy to `.env.production`)

Never commit real `.env*` files.

## Option 1: Bare Binary / systemd

Build (no CGO, no system libraries needed):

```bash
make build                # produces bin/shopogoda
# or: CGO_ENABLED=0 go build -o shopogoda cmd/bot/main.go
```

Install and run:

```bash
sudo useradd --system --home /var/lib/shopogoda --create-home shopogoda
sudo install -m 0755 bin/shopogoda /usr/local/bin/shopogoda
sudo install -d -o shopogoda -g shopogoda /var/lib/shopogoda/data
sudo install -m 0600 -o root -g root .env.production /etc/shopogoda.env
```

`/etc/systemd/system/shopogoda.service`:

```ini
[Unit]
Description=ShoPogoda Telegram bot
After=network-online.target
Wants=network-online.target

[Service]
User=shopogoda
WorkingDirectory=/var/lib/shopogoda
EnvironmentFile=/etc/shopogoda.env
Environment=DB_PATH=/var/lib/shopogoda/data/shopogoda.db
ExecStart=/usr/local/bin/shopogoda
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now shopogoda
journalctl -u shopogoda -f
```

The schema is created/migrated automatically on startup. `make migrate` runs the
same AutoMigrate step standalone.

## Option 2: Docker

Images are published to `ghcr.io/valpere/shopogoda`. To build locally:

```bash
make docker-build
```

Run with a named volume for the database:

```bash
docker run -d --name shopogoda --restart unless-stopped \
  --env-file .env.production \
  -p 8080:8080 \
  -v shopogoda_data:/app/data \
  ghcr.io/valpere/shopogoda:latest
```

The image sets `DB_PATH=/app/data/shopogoda.db` and declares `VOLUME /app/data`.
Without a mounted volume the database is lost when the container is removed.

## Option 3: Docker Compose (staging / production)

`docker/docker-compose.prod.yml` and `docker/docker-compose.staging.yml` define
the `bot` service plus optional observability containers (Prometheus, Grafana,
Jaeger). The bot alone is enough; remove the others if you do not need them.

| | Staging | Production |
|-|---------|------------|
| Env file | `.env.staging` | `.env.production` |
| Bot container | `shopogoda-bot-staging` | `shopogoda-bot-prod` |
| Data volume | `shopogoda_data_staging` | `shopogoda_data_prod` |
| Host port -> 8080 | `8081` | `8080` (`BOT_WEBHOOK_PORT`) |

Production requires `GRAFANA_ADMIN_PASSWORD` to be set (Grafana service).

```bash
cp .env.prod.example .env.production      # edit values
docker compose -f docker/docker-compose.prod.yml up -d
docker compose -f docker/docker-compose.prod.yml ps
docker compose -f docker/docker-compose.prod.yml logs -f bot
```

### Helper scripts

`scripts/deploy/deploy.sh` wraps the compose workflow (it uses the
`docker-compose` command and expects `.env.<environment>` in the project root):

```bash
./scripts/deploy/deploy.sh staging develop
./scripts/deploy/deploy.sh production v1.2.3
./scripts/deploy/health-check.sh production
./scripts/deploy/rollback.sh production
```

`deploy.sh` saves a snapshot (container list, image list, a copy of the env
file) under `backups/<timestamp>/`, pulls images, runs `up -d` and waits for the
containers to report healthy. `rollback.sh` restores the saved env file and
restarts the stack. **These snapshots do not contain the SQLite database**; take
a database backup separately (see below).

## Option 4: Kubernetes

Manifests are in `deployments/k8s/`. Because SQLite allows a single writer, the
Deployment uses `replicas: 1` and `strategy: Recreate`, and the database sits on
a PersistentVolumeClaim (`shopogoda-data`, `ReadWriteOnce`, 1Gi) mounted at
`/app/data`. Do not scale beyond one replica and do not switch to
`RollingUpdate`.

1. Edit `deployments/k8s/secret.yaml` (`TELEGRAM_BOT_TOKEN`,
   `OPENWEATHER_API_KEY`, optional `SLACK_WEBHOOK_URL`) and `configmap.yaml`
   (`DB_PATH`, `BOT_WEBHOOK_PORT`, `LOG_*`). Set `BOT_WEBHOOK_URL` there too if
   you use webhooks.
2. Make the image available to the cluster (the manifest references
   `shopogoda:latest`; change it to your registry tag, e.g.
   `ghcr.io/valpere/shopogoda:<version>`).
3. Apply:

```bash
kubectl apply -f deployments/k8s/namespace.yaml
kubectl apply -f deployments/k8s/pvc.yaml
kubectl apply -f deployments/k8s/configmap.yaml
kubectl apply -f deployments/k8s/secret.yaml
kubectl apply -f deployments/k8s/deployment.yaml
kubectl apply -f deployments/k8s/service.yaml
kubectl -n shopogoda rollout status deployment/shopogoda
```

The Service (`shopogoda-service`, ClusterIP) exposes port 8080; expose it through
your Ingress for webhook mode. Liveness and readiness probes use `GET /health`.
The container runs as the non-root `shopogoda` user (uid/gid 10001); the pod spec
sets `runAsUser`/`runAsGroup`/`fsGroup` to 10001 so the PVC is writable for SQLite.

## Webhook vs Polling, Reverse Proxy and HTTPS

- **Polling** (leave `BOT_WEBHOOK_URL` empty): the bot pulls updates from
  Telegram. No public URL or inbound port is needed. Fine for development and
  small deployments.
- **Webhook** (set `BOT_WEBHOOK_URL=https://bot.example.com`; the bot registers
  `<URL>/webhook`): Telegram requires a public **HTTPS** endpoint, so put a
  TLS-terminating reverse proxy in front of port 8080.

Caddy (automatic certificates):

```
bot.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

nginx (certificates via certbot or your own):

```nginx
server {
    listen 443 ssl;
    server_name bot.example.com;
    # ssl_certificate / ssl_certificate_key ...

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
    }
}
```

Only `/webhook` and `/health` need to be public. Do not expose `/metrics` or
the Prometheus/Grafana/Jaeger ports to the internet.

## Data, Volumes and Permissions

The database file is `DB_PATH` plus, while the bot runs, `-wal` and `-shm`
sidecar files. Always keep the three together in one directory.

The container runs as the non-root user `shopogoda`, which owns `/app/data`
inside the image. Named Docker volumes are initialised from that directory and
work as-is. **Bind mounts do not**: the host directory must be writable by the
container user's uid (fixed at 10001:10001 in the Dockerfile). Fix ownership:

```bash
sudo chown -R 10001:10001 /srv/shopogoda/data
```

Locate a named volume on the host with `docker volume inspect shopogoda_data_prod`.
Place the database on local disk; avoid network filesystems (NFS/SMB), where
SQLite file locking is unreliable.

## Backups and Restore

The `sqlite3` CLI is not in the application image; run it on the host, or use a
throwaway container.

**Online backup (safe while the bot is running, WAL mode):**

```bash
# bare binary / host path
sqlite3 /var/lib/shopogoda/data/shopogoda.db ".backup /backups/shopogoda-$(date +%F).db"

# named Docker volume (no sqlite3 on the host needed)
mkdir -p backups
docker run --rm -v shopogoda_data_prod:/data -v "$PWD/backups":/backup \
  keinos/sqlite3 sqlite3 /data/shopogoda.db ".backup /backup/shopogoda-$(date +%F).db"
```

Do not copy the `.db` file alone with `cp` while the bot is running; the latest
writes may still be in the `-wal` file. For continuous replication, consider
[Litestream](https://litestream.io/) as a sidecar or host service.

Run backups from cron or a systemd timer and copy them off the host. Test a
restore occasionally.

**Restore:**

1. Stop the bot (`systemctl stop shopogoda`, `docker compose ... stop bot`, or
   scale the Kubernetes deployment to 0).
2. Replace the database file with the backup and **delete the stale
   `shopogoda.db-wal` and `shopogoda.db-shm`** next to it.
3. Fix ownership (`shopogoda` user) if you copied as root.
4. Start the bot and check `/health` and the logs.

```bash
docker compose -f docker/docker-compose.prod.yml stop bot
docker run --rm -v shopogoda_data_prod:/data -v "$PWD/backups":/backup alpine \
  sh -c 'rm -f /data/shopogoda.db-wal /data/shopogoda.db-shm && \
         cp /backup/shopogoda-2026-01-01.db /data/shopogoda.db && \
         chown 10001:10001 /data/shopogoda.db'
docker compose -f docker/docker-compose.prod.yml start bot
```

## Upgrades and Rollback

Single instance means a brief outage per upgrade (seconds); Telegram retries
undelivered webhook updates, and in polling mode updates queue on Telegram's side.

1. Take a database backup.
2. Stop -> pull -> start:

```bash
# Docker Compose
VERSION=v1.2.4 docker compose -f docker/docker-compose.prod.yml pull bot
VERSION=v1.2.4 docker compose -f docker/docker-compose.prod.yml up -d bot

# systemd
sudo systemctl stop shopogoda
sudo install -m 0755 bin/shopogoda /usr/local/bin/shopogoda
sudo systemctl start shopogoda

# Kubernetes
kubectl -n shopogoda set image deployment/shopogoda shopogoda=ghcr.io/valpere/shopogoda:v1.2.4
```

3. Verify `/health` and logs. Schema changes are applied automatically on
   startup (GORM AutoMigrate).

**Rollback:** redeploy the previous image tag/binary (`./scripts/deploy/rollback.sh` for compose setups). If the new version
changed the schema in a way the old version cannot read, also restore the
pre-upgrade database backup. The in-memory cache starts empty after every
restart, which is expected.

## Health Checks and Metrics

- **Health:** `GET /health` on the HTTP port (8080) returns
  `{"status":"healthy",...}`. The Docker `HEALTHCHECK` and the Kubernetes probes
  use it. `./scripts/deploy/health-check.sh [staging|production]` checks
  container state, endpoints and that the database file exists.
- **Metrics:** Prometheus metrics are served at `GET /metrics` on the HTTP
  server (port 8080) of the bot; the bundled `deployments/prometheus.yml`
  scrapes `host.docker.internal:8080`. `PROMETHEUS_PORT` (2112) is read into
  configuration but nothing binds it.
- **Observability stack (optional):** Prometheus `:9090`, Grafana `:3000`,
  Jaeger `:16686`. For local development `make docker-up` starts only this
  stack.
- **Logs:** structured JSON on stdout (`LOG_FORMAT=json`);
  `docker compose logs -f bot`, `journalctl -u shopogoda -f`, or
  `kubectl -n shopogoda logs -f deployment/shopogoda`.

Admin statistics (messages, weather requests) are in-memory counters labelled
"since start" and reset on restart.

## Troubleshooting

**Container or service will not start**

```bash
docker compose -f docker/docker-compose.prod.yml logs bot
docker compose -f docker/docker-compose.prod.yml config   # check resolved env
```

Check that `TELEGRAM_BOT_TOKEN` and `OPENWEATHER_API_KEY` are set.

**"unable to open database file" / "attempt to write a readonly database"**

The process user cannot write to the data directory. Fix ownership of the
directory (and of `-wal`/`-shm` files) as described in
[Data, Volumes and Permissions](#data-volumes-and-permissions).

**"database is locked"**

More than one process is using the same database file (a second container, a
leftover process, or a `sqlite3` shell holding a write lock). Run a single
instance and stop the extra one.

**Webhook not receiving updates**

Confirm the public URL is HTTPS with a valid certificate, that `BOT_WEBHOOK_URL`
matches it, and check Telegram's view:
`curl https://api.telegram.org/bot<TOKEN>/getWebhookInfo`. Alternatively leave
`BOT_WEBHOOK_URL` empty to use polling.

**Port already in use**

```bash
sudo lsof -i :8080
```

Change `BOT_WEBHOOK_PORT` (compose host port) or stop the conflicting service.

**Out of disk space**

```bash
df -h
docker system df
docker image prune -f
```

**Debug mode**

Set `BOT_DEBUG=true` and `LOG_LEVEL=debug` temporarily, restart, then revert.
Never leave debug on in production.

## Security Notes

- Keep `.env.production` out of git; restrict file permissions (`0600`).
- Run as the non-root user; do not publish observability ports publicly.
- Use HTTPS only for the webhook.
- Rotate the Telegram token and API keys periodically.
- Restrict access to the database file and to backups (they contain user data).

## Additional Resources

- [Configuration Reference](CONFIGURATION.md)
- [Admin Setup](ADMIN_SETUP.md)
- [Release Process](RELEASE_PROCESS.md)
- [Architecture](ARCHITECTURE.md)
- [Development Guide](../README.md)
- Issues: <https://github.com/valpere/shopogoda/issues>
