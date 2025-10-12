# Google Cloud Platform Deployment Guide

Complete guide for deploying ShoPogoda to Google Cloud Platform using free tier services.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Cost Estimation](#cost-estimation)
- [Prerequisites](#prerequisites)
- [Quick Start (5 Minutes)](#quick-start-5-minutes)
- [Detailed Setup](#detailed-setup)
- [Configuration](#configuration)
- [Deployment Methods](#deployment-methods)
- [Monitoring & Maintenance](#monitoring--maintenance)
- [Troubleshooting](#troubleshooting)

## Architecture Overview

### Recommended Architecture (Free Tier Optimized)

```plaintext
┌─────────────────────────────────────────────────────────────┐
│                     GCP Deployment                          │
├─────────────────────────────────────────────────────────────┤
│  Cloud Run (Bot Application)                                │
│  ├── Serverless, scales to zero                             │
│  ├── HTTPS endpoint for Telegram webhook                    │
│  ├── Health checks at /health                               │
│  └── Auto-scaling based on requests                         │
├─────────────────────────────────────────────────────────────┤
│  Cloud SQL for PostgreSQL                                   │
│  ├── Managed PostgreSQL 15                                  │
│  ├── Automatic backups                                      │
│  ├── Private IP or Cloud SQL Proxy                          │
│  └── db-f1-micro instance (within free trial)               │
├─────────────────────────────────────────────────────────────┤
│  Redis Options                                              │
│  ├── Option 1: Memorystore Redis (paid, $25/month)          │
│  ├── Option 2: Upstash Redis (free tier, recommended)       │
│  └── Option 3: Redis on Compute Engine (free e2-micro)      │
├─────────────────────────────────────────────────────────────┤
│  Supporting Services                                        │
│  ├── Artifact Registry (container images)                   │
│  ├── Cloud Build (CI/CD)                                    │
│  ├── Secret Manager (API keys)                              │
│  └── Cloud Logging & Monitoring                             │
└─────────────────────────────────────────────────────────────┘
```

### Free Tier Optimization

**Services Used:**

- ✅ **Cloud Run**: Free tier (2M requests/month)
- ✅ **Cloud SQL**: Free trial credit ($300 for 90 days)
- ✅ **Artifact Registry**: Free (0.5 GB)
- ✅ **Cloud Build**: Free (120 build-minutes/day)
- ⚠️ **Redis**: Use external Upstash (free tier)

## Cost Estimation

### Month 1-3 (Free Trial with $300 credit)

- Cloud Run: $0 (within free tier)
- Cloud SQL (db-f1-micro): ~$15/month (paid from free credits)
- Artifact Registry: $0 (within free tier)
- Redis (Upstash free): $0
- **Total: $0** (using free credits)

### After Free Trial (Month 4+)

- Cloud Run: $0-2/month (likely $0 with webhook mode)
- Cloud SQL (db-f1-micro): $15/month
- Redis (Upstash free): $0
- **Total: ~$15/month**

### Cost Optimization Options:

1. **Use SQLite instead of Cloud SQL**: $0/month (store in Cloud Storage)
2. **Use Firestore for user data**: $0/month (within free tier)
3. **Self-hosted Redis on e2-micro**: $0/month (within always-free tier)

## Prerequisites

### 1. GCP Account Setup

```bash
# 1. Create GCP account (if not done)
# Visit: https://console.cloud.google.com

# 2. Enable billing (required even for free tier)
# Add payment method (won't charge without explicit upgrade)

# 3. Create new project
# Project ID: shopogoda-bot (or your choice)
```

### 2. Install Google Cloud CLI

**Linux:**

```bash
# Download and install
curl https://sdk.cloud.google.com | bash

# Restart shell
exec -l $SHELL

# Initialize
gcloud init
```

**macOS:**

```bash
# Install via Homebrew
brew install google-cloud-sdk

# Initialize
gcloud init
```

**Authentication:**

```bash
# Login to GCP
gcloud auth login

# Set project
gcloud config set project shopogoda-bot

# Enable required APIs
gcloud services enable \
  run.googleapis.com \
  sql-component.googleapis.com \
  sqladmin.googleapis.com \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  secretmanager.googleapis.com
```

### 3. Get API Keys

**Telegram Bot Token:**

```bash
# Message @BotFather on Telegram:
/newbot
# Save the token (format: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz)
```

**OpenWeatherMap API Key:**

```bash
# Sign up at https://openweathermap.org/api
# Free tier: 60 calls/minute, 1M calls/month
# Copy API key from dashboard
```

## Quick Start (5 Minutes)

### One-Command Deployment (Recommended)

```bash
# Clone repository
git clone https://github.com/valpere/shopogoda.git
cd shopogoda

# Run deployment script
./scripts/gcp-deploy.sh
```

The script will:

1. Create Cloud SQL PostgreSQL instance
2. Set up Artifact Registry
3. Build and push Docker image
4. Deploy to Cloud Run
5. Configure webhook
6. Display bot URL

**Note:** You'll be prompted for:

- GCP Project ID
- Telegram Bot Token
- OpenWeatherMap API Key
- Redis connection (Upstash recommended)

## Detailed Setup

### Step 1: Create Cloud SQL PostgreSQL Instance

```bash
# Set variables
export PROJECT_ID="shopogoda-bot"
export REGION="us-central1"  # Choose nearest region
export DB_INSTANCE="shopogoda-db"
export DB_NAME="shopogoda"
export DB_USER="shopogoda_user"

# Generate secure password
export DB_PASSWORD=$(openssl rand -base64 32)
echo "Database Password: $DB_PASSWORD" > ~/shopogoda-db-password.txt
echo "⚠️  Save this password securely!"

# Create Cloud SQL instance (db-f1-micro for free trial)
gcloud sql instances create $DB_INSTANCE \
  --database-version=POSTGRES_15 \
  --tier=db-f1-micro \
  --region=$REGION \
  --root-password="$DB_PASSWORD" \
  --backup \
  --backup-start-time=03:00

# Create database
gcloud sql databases create $DB_NAME \
  --instance=$DB_INSTANCE

# Create user
gcloud sql users create $DB_USER \
  --instance=$DB_INSTANCE \
  --password="$DB_PASSWORD"

# Get connection name
export DB_CONNECTION_NAME=$(gcloud sql instances describe $DB_INSTANCE --format='get(connectionName)')
echo "Connection Name: $DB_CONNECTION_NAME"
```

### Step 2: Set Up Redis (Upstash Free Tier)

**Why Upstash:**

- Free tier: 10,000 commands/day
- Global low-latency
- Serverless (pay only for usage)
- Perfect for Cloud Run

**Setup:**

```bash
# 1. Sign up at https://upstash.com (free, no credit card)
# 2. Create database:
#    - Name: shopogoda-cache
#    - Region: Choose nearest to your Cloud Run region
#    - Type: Regional
# 3. Copy connection details:
#    - Endpoint (e.g., us1-adapted-bat-12345.upstash.io)
#    - Port (6379)
#    - Password

# Set as environment variables
export REDIS_HOST="us1-adapted-bat-12345.upstash.io"
export REDIS_PORT="6379"
export REDIS_PASSWORD="your_upstash_password"
```

**Alternative: Self-Hosted Redis on e2-micro (Free Tier)**

```bash
# Create e2-micro instance (always free)
gcloud compute instances create shopogoda-redis \
  --machine-type=e2-micro \
  --zone=us-central1-a \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=10GB

# SSH and install Redis
gcloud compute ssh shopogoda-redis --zone=us-central1-a

# On the VM:
sudo apt update
sudo apt install -y redis-server
sudo sed -i 's/bind 127.0.0.1/bind 0.0.0.0/' /etc/redis/redis.conf
sudo systemctl restart redis
exit

# Get internal IP
export REDIS_HOST=$(gcloud compute instances describe shopogoda-redis --zone=us-central1-a --format='get(networkInterfaces[0].networkIP)')
```

### Step 3: Store Secrets in Secret Manager

```bash
# Create secrets
echo -n "YOUR_TELEGRAM_BOT_TOKEN" | \
  gcloud secrets create telegram-bot-token --data-file=-

echo -n "YOUR_OPENWEATHER_API_KEY" | \
  gcloud secrets create openweather-api-key --data-file=-

echo -n "$DB_PASSWORD" | \
  gcloud secrets create db-password --data-file=-

echo -n "$REDIS_PASSWORD" | \
  gcloud secrets create redis-password --data-file=-

# Grant Cloud Run access to secrets
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:PROJECT_NUMBER-compute@developer.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### Step 4: Create Artifact Registry

```bash
# Create Docker repository
gcloud artifacts repositories create shopogoda \
  --repository-format=docker \
  --location=$REGION \
  --description="ShoPogoda container images"

# Configure Docker authentication
gcloud auth configure-docker ${REGION}-docker.pkg.dev
```

### Step 5: Build and Push Docker Image

```bash
# Build image
docker build -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:latest .

# Push to Artifact Registry
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:latest
```

**Using Cloud Build (Recommended - Free 120 min/day):**

```bash
# Build in cloud (no local Docker needed)
gcloud builds submit \
  --tag ${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:latest

# Or use cloudbuild.yaml for CI/CD
gcloud builds submit --config cloudbuild.yaml
```

### Step 6: Deploy to Cloud Run

```bash
# Deploy service
gcloud run deploy shopogoda-bot \
  --image=${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:latest \
  --region=$REGION \
  --platform=managed \
  --allow-unauthenticated \
  --port=8080 \
  --memory=512Mi \
  --cpu=1 \
  --min-instances=0 \
  --max-instances=10 \
  --timeout=300 \
  --set-env-vars="LOG_LEVEL=info,BOT_DEBUG=false" \
  --set-secrets="TELEGRAM_BOT_TOKEN=telegram-bot-token:latest,OPENWEATHER_API_KEY=openweather-api-key:latest,DB_PASSWORD=db-password:latest,REDIS_PASSWORD=redis-password:latest" \
  --set-env-vars="DB_HOST=/cloudsql/$DB_CONNECTION_NAME,DB_PORT=5432,DB_NAME=$DB_NAME,DB_USER=$DB_USER,REDIS_HOST=$REDIS_HOST,REDIS_PORT=$REDIS_PORT" \
  --add-cloudsql-instances=$DB_CONNECTION_NAME

# Get service URL
export SERVICE_URL=$(gcloud run services describe shopogoda-bot --region=$REGION --format='get(status.url)')
echo "Bot deployed at: $SERVICE_URL"
```

### Step 7: Configure Telegram Webhook

```bash
# Set webhook
curl -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  -d "url=${SERVICE_URL}/webhook"

# Verify webhook
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getWebhookInfo"
```

## Configuration

### Environment Variables

**Required:**

```bash
TELEGRAM_BOT_TOKEN     # From Secret Manager
OPENWEATHER_API_KEY    # From Secret Manager
DB_HOST                # Cloud SQL connection
DB_PORT                # 5432
DB_NAME                # shopogoda
DB_USER                # shopogoda_user
DB_PASSWORD            # From Secret Manager
REDIS_HOST             # Upstash or e2-micro IP
REDIS_PORT             # 6379
REDIS_PASSWORD         # From Secret Manager (if using Upstash)
```

**Optional:**

```bash
BOT_WEBHOOK_URL        # ${SERVICE_URL}/webhook
LOG_LEVEL              # info
BOT_DEBUG              # false
SLACK_WEBHOOK_URL      # Enterprise notifications
```

### Cloud Run Service Configuration

**cloudbuild.yaml** (for automated deployments):

```yaml
steps:
  # Build Docker image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'build'
      - '-t'
      - '${_REGION}-docker.pkg.dev/${PROJECT_ID}/${_REPO_NAME}/bot:${SHORT_SHA}'
      - '-t'
      - '${_REGION}-docker.pkg.dev/${PROJECT_ID}/${_REPO_NAME}/bot:latest'
      - '.'

  # Push to Artifact Registry
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - '${_REGION}-docker.pkg.dev/${PROJECT_ID}/${_REPO_NAME}/bot:${SHORT_SHA}'

  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - '${_REGION}-docker.pkg.dev/${PROJECT_ID}/${_REPO_NAME}/bot:latest'

  # Deploy to Cloud Run
  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: gcloud
    args:
      - 'run'
      - 'deploy'
      - 'shopogoda-bot'
      - '--image=${_REGION}-docker.pkg.dev/${PROJECT_ID}/${_REPO_NAME}/bot:${SHORT_SHA}'
      - '--region=${_REGION}'
      - '--platform=managed'
      - '--allow-unauthenticated'

substitutions:
  _REGION: us-central1
  _REPO_NAME: shopogoda

options:
  logging: CLOUD_LOGGING_ONLY
```

## Deployment Methods

### Method 1: Manual Deployment (Current)

```bash
# Build locally and push
docker build -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:v1.0.0 .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:v1.0.0

# Deploy specific version
gcloud run deploy shopogoda-bot \
  --image=${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:v1.0.0 \
  --region=$REGION
```

### Method 2: Cloud Build (Automated)

```bash
# Deploy from source (Cloud Build handles everything)
gcloud run deploy shopogoda-bot \
  --source . \
  --region=$REGION

# Or trigger build from cloudbuild.yaml
gcloud builds submit --config cloudbuild.yaml
```

### Method 3: GitHub Actions CI/CD

Create `.github/workflows/deploy-gcp.yml`:

```yaml
name: Deploy to GCP Cloud Run

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - id: auth
        uses: google-github-actions/auth@v2
        with:
          credentials_json: ${{ secrets.GCP_SA_KEY }}

      - name: Set up Cloud SDK
        uses: google-github-actions/setup-gcloud@v2

      - name: Build and push
        run: |
          gcloud builds submit \
            --tag ${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:${{ github.sha }}

      - name: Deploy to Cloud Run
        run: |
          gcloud run deploy shopogoda-bot \
            --image ${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:${{ github.sha }} \
            --region ${REGION}
```

## Monitoring & Maintenance

### Health Checks

```bash
# Check bot health
curl https://shopogoda-bot-xxxxx.run.app/health

# View Cloud Run logs
gcloud run services logs read shopogoda-bot \
  --region=$REGION \
  --limit=50

# Stream logs in real-time
gcloud run services logs tail shopogoda-bot --region=$REGION
```

### Cloud Logging Queries

```bash
# View all logs in Cloud Console
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=shopogoda-bot" --limit=100

# Filter errors only
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=shopogoda-bot AND severity>=ERROR" --limit=50
```

### Monitoring Dashboard

**Cloud Console:**

1. Go to Cloud Run → shopogoda-bot
2. View tabs: Metrics, Logs, Revisions
3. Monitor: Request count, latency, error rate, CPU/Memory usage

**Set Up Alerts:**

```bash
# Create alert for error rate
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="ShoPogoda Error Rate Alert" \
  --condition-display-name="Error rate > 5%" \
  --condition-threshold-value=5 \
  --condition-threshold-duration=300s
```

### Database Backups

```bash
# Cloud SQL automatic backups enabled by default

# Manual backup
gcloud sql backups create --instance=$DB_INSTANCE

# List backups
gcloud sql backups list --instance=$DB_INSTANCE

# Restore from backup
gcloud sql backups restore BACKUP_ID --backup-instance=$DB_INSTANCE
```

## Troubleshooting

### Common Issues

**1. Cloud Run service won't start**

```bash
# Check logs
gcloud run services logs read shopogoda-bot --region=$REGION --limit=100

# Common causes:
# - Missing environment variables
# - Database connection failed
# - Port mismatch (ensure app listens on PORT env var)
```

**2. Database connection timeout**

```bash
# Verify Cloud SQL connection
gcloud sql instances describe $DB_INSTANCE

# Check if Cloud SQL API is enabled
gcloud services list --enabled | grep sql

# Ensure Cloud Run has --add-cloudsql-instances flag
gcloud run services describe shopogoda-bot --region=$REGION
```

**3. High costs**

```bash
# Check billing
gcloud billing accounts list

# View current month costs
gcloud billing accounts get-billing-info

# Optimize:
# - Set min-instances=0 (scale to zero)
# - Use smaller Cloud SQL tier (db-f1-micro)
# - Enable request-based scaling
```

**4. Webhook not receiving updates**

```bash
# Check webhook status
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getWebhookInfo"

# Reset webhook
curl -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/deleteWebhook"
curl -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  -d "url=${SERVICE_URL}/webhook"
```

### Debug Mode

```bash
# Enable debug logging
gcloud run services update shopogoda-bot \
  --region=$REGION \
  --set-env-vars="LOG_LEVEL=debug,BOT_DEBUG=true"

# View detailed logs
gcloud run services logs tail shopogoda-bot --region=$REGION
```

### Performance Optimization

**Reduce cold starts:**

```bash
# Set minimum instances (costs more but faster)
gcloud run services update shopogoda-bot \
  --region=$REGION \
  --min-instances=1
```

**Increase resources:**

```bash
# For high-traffic bots
gcloud run services update shopogoda-bot \
  --region=$REGION \
  --memory=1Gi \
  --cpu=2
```

## Cost Optimization Tips

### 1. Free Tier Maximization

```bash
# Use smallest Cloud SQL instance
--tier=db-f1-micro  # $15/month (covered by free trial)

# Scale Cloud Run to zero
--min-instances=0  # No cost when idle

# Use Upstash Redis free tier
# 10,000 commands/day free
```

### 2. Alternative: Serverless Architecture (Lower Cost)

**Use Firestore instead of Cloud SQL:**

- Free tier: 1 GB storage, 50K reads, 20K writes/day
- No database instance costs
- Requires code changes to use NoSQL

**Use Cloud Storage for SQLite:**

- Free tier: 5 GB storage
- Mount SQLite file from Cloud Storage
- No database costs at all

### 3. Budget Alerts

```bash
# Set budget alert
gcloud billing budgets create \
  --billing-account=BILLING_ACCOUNT_ID \
  --display-name="ShoPogoda Monthly Budget" \
  --budget-amount=20 \
  --threshold-rule=percent=90
```

## Next Steps

1. ✅ Deploy bot to Cloud Run
2. ✅ Configure webhook
3. ✅ Set up monitoring
4. 🔄 Test bot functionality
5. 🔄 Configure domain (optional): `bot.yourdomain.com`
6. 🔄 Set up CI/CD with GitHub Actions
7. 🔄 Configure backup strategy

## Resources

- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud SQL PostgreSQL](https://cloud.google.com/sql/docs/postgres)
- [Artifact Registry](https://cloud.google.com/artifact-registry/docs)
- [Secret Manager](https://cloud.google.com/secret-manager/docs)
- [GCP Free Tier](https://cloud.google.com/free)

---

**Last Updated**: 2025-01-03
**Maintained by**: [@valpere](https://github.com/valpere)
