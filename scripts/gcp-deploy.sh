#!/bin/bash

# ShoPogoda GCP Deployment Script
# Quick deployment to Google Cloud Run with Cloud SQL and Upstash Redis

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ShoPogoda - GCP Deployment Script           ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
command -v gcloud >/dev/null 2>&1 || { echo -e "${RED}Error: gcloud CLI not found. Install from https://cloud.google.com/sdk${NC}"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo -e "${RED}Error: docker not found. Install from https://docs.docker.com/get-docker/${NC}"; exit 1; }

# Get user inputs
echo -e "${YELLOW}Please provide the following information:${NC}"
echo ""

read -p "GCP Project ID [shopogoda-bot]: " PROJECT_ID
PROJECT_ID=${PROJECT_ID:-shopogoda-bot}

read -p "Region [us-central1]: " REGION
REGION=${REGION:-us-central1}

read -p "Telegram Bot Token: " TELEGRAM_BOT_TOKEN
if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
    echo -e "${RED}Error: Telegram Bot Token is required${NC}"
    exit 1
fi

read -p "OpenWeatherMap API Key: " OPENWEATHER_API_KEY
if [ -z "$OPENWEATHER_API_KEY" ]; then
    echo -e "${RED}Error: OpenWeatherMap API Key is required${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}Redis Setup:${NC}"
echo "1. Upstash Redis (Free tier, recommended)"
echo "2. Self-hosted on e2-micro (GCP always-free)"
echo "3. Skip (configure later)"
read -p "Choose Redis option [1]: " REDIS_OPTION
REDIS_OPTION=${REDIS_OPTION:-1}

if [ "$REDIS_OPTION" = "1" ]; then
    echo ""
    echo -e "${YELLOW}Sign up for free at: https://upstash.com${NC}"
    echo "Create a database and get connection details:"
    read -p "Upstash Redis Host (e.g., us1-adapted-bat-12345.upstash.io): " REDIS_HOST
    read -p "Upstash Redis Port [6379]: " REDIS_PORT
    REDIS_PORT=${REDIS_PORT:-6379}
    read -p "Upstash Redis Password: " REDIS_PASSWORD
fi

echo ""
echo -e "${GREEN}Configuration Summary:${NC}"
echo "  Project ID: $PROJECT_ID"
echo "  Region: $REGION"
echo "  Telegram Token: ${TELEGRAM_BOT_TOKEN:0:10}..."
echo "  Weather API Key: ${OPENWEATHER_API_KEY:0:10}..."
if [ "$REDIS_OPTION" = "1" ]; then
    echo "  Redis: Upstash ($REDIS_HOST)"
elif [ "$REDIS_OPTION" = "2" ]; then
    echo "  Redis: Self-hosted e2-micro"
else
    echo "  Redis: Skipped (manual setup required)"
fi
echo ""

read -p "Continue with deployment? [Y/n]: " CONFIRM
CONFIRM=${CONFIRM:-Y}
if [ "$CONFIRM" != "Y" ] && [ "$CONFIRM" != "y" ]; then
    echo "Deployment cancelled."
    exit 0
fi

echo ""
echo -e "${GREEN}Starting deployment...${NC}"
echo ""

# Set project
echo -e "${YELLOW}[1/8] Setting up GCP project...${NC}"
gcloud config set project $PROJECT_ID

# Enable required APIs
echo -e "${YELLOW}[2/8] Enabling GCP APIs...${NC}"
gcloud services enable \
  run.googleapis.com \
  sql-component.googleapis.com \
  sqladmin.googleapis.com \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  secretmanager.googleapis.com \
  --quiet

# Create secrets
echo -e "${YELLOW}[3/8] Storing secrets in Secret Manager...${NC}"
echo -n "$TELEGRAM_BOT_TOKEN" | gcloud secrets create telegram-bot-token --data-file=- --replication-policy=automatic 2>/dev/null || \
  echo -n "$TELEGRAM_BOT_TOKEN" | gcloud secrets versions add telegram-bot-token --data-file=-

echo -n "$OPENWEATHER_API_KEY" | gcloud secrets create openweather-api-key --data-file=- --replication-policy=automatic 2>/dev/null || \
  echo -n "$OPENWEATHER_API_KEY" | gcloud secrets versions add openweather-api-key --data-file=-

# Generate database password
DB_PASSWORD=$(openssl rand -base64 32)
echo -n "$DB_PASSWORD" | gcloud secrets create db-password --data-file=- --replication-policy=automatic 2>/dev/null || \
  echo -n "$DB_PASSWORD" | gcloud secrets versions add db-password --data-file=-

# Store Redis password if using Upstash
if [ "$REDIS_OPTION" = "1" ] && [ -n "$REDIS_PASSWORD" ]; then
    echo -n "$REDIS_PASSWORD" | gcloud secrets create redis-password --data-file=- --replication-policy=automatic 2>/dev/null || \
      echo -n "$REDIS_PASSWORD" | gcloud secrets versions add redis-password --data-file=-
fi

# Create Cloud SQL instance
echo -e "${YELLOW}[4/8] Creating Cloud SQL PostgreSQL instance (this may take 5-10 minutes)...${NC}"
DB_INSTANCE="shopogoda-db"

if ! gcloud sql instances describe $DB_INSTANCE --quiet 2>/dev/null; then
    gcloud sql instances create $DB_INSTANCE \
      --database-version=POSTGRES_15 \
      --tier=db-f1-micro \
      --region=$REGION \
      --root-password="$DB_PASSWORD" \
      --backup \
      --backup-start-time=03:00 \
      --quiet

    # Create database and user
    gcloud sql databases create shopogoda --instance=$DB_INSTANCE --quiet
    gcloud sql users create shopogoda_user --instance=$DB_INSTANCE --password="$DB_PASSWORD" --quiet
else
    echo "  Instance $DB_INSTANCE already exists, skipping creation"
fi

DB_CONNECTION_NAME=$(gcloud sql instances describe $DB_INSTANCE --format='get(connectionName)')
echo "  Connection Name: $DB_CONNECTION_NAME"

# Setup Redis if option 2 (self-hosted)
if [ "$REDIS_OPTION" = "2" ]; then
    echo -e "${YELLOW}Setting up self-hosted Redis on e2-micro...${NC}"

    if ! gcloud compute instances describe shopogoda-redis --zone=${REGION}-a --quiet 2>/dev/null; then
        gcloud compute instances create shopogoda-redis \
          --machine-type=e2-micro \
          --zone=${REGION}-a \
          --image-family=ubuntu-2204-lts \
          --image-project=ubuntu-os-cloud \
          --boot-disk-size=10GB \
          --metadata=startup-script='#!/bin/bash
            apt update
            apt install -y redis-server
            sed -i "s/bind 127.0.0.1/bind 0.0.0.0/" /etc/redis/redis.conf
            systemctl restart redis' \
          --quiet

        echo "  Waiting for Redis to start..."
        sleep 30
    fi

    REDIS_HOST=$(gcloud compute instances describe shopogoda-redis --zone=${REGION}-a --format='get(networkInterfaces[0].networkIP)')
    REDIS_PORT="6379"
    echo "  Redis Host: $REDIS_HOST"
fi

# Create Artifact Registry repository
echo -e "${YELLOW}[5/8] Setting up Artifact Registry...${NC}"
gcloud artifacts repositories create shopogoda \
  --repository-format=docker \
  --location=$REGION \
  --description="ShoPogoda container images" \
  --quiet 2>/dev/null || echo "  Repository already exists"

gcloud auth configure-docker ${REGION}-docker.pkg.dev --quiet

# Build and push image
echo -e "${YELLOW}[6/8] Building Docker image with Cloud Build...${NC}"
IMAGE_TAG="${REGION}-docker.pkg.dev/${PROJECT_ID}/shopogoda/bot:latest"

gcloud builds submit \
  --tag $IMAGE_TAG \
  --timeout=15m \
  --quiet

# Deploy to Cloud Run
echo -e "${YELLOW}[7/8] Deploying to Cloud Run...${NC}"

DEPLOY_CMD="gcloud run deploy shopogoda-bot \
  --image=$IMAGE_TAG \
  --region=$REGION \
  --platform=managed \
  --allow-unauthenticated \
  --port=8080 \
  --memory=512Mi \
  --cpu=1 \
  --min-instances=0 \
  --max-instances=10 \
  --timeout=300 \
  --set-env-vars=LOG_LEVEL=info,BOT_DEBUG=false,DB_PORT=5432,DB_NAME=shopogoda,DB_USER=shopogoda_user \
  --set-secrets=TELEGRAM_BOT_TOKEN=telegram-bot-token:latest,OPENWEATHER_API_KEY=openweather-api-key:latest,DB_PASSWORD=db-password:latest \
  --add-cloudsql-instances=$DB_CONNECTION_NAME \
  --quiet"

# Add Redis configuration based on option
if [ "$REDIS_OPTION" = "1" ] || [ "$REDIS_OPTION" = "2" ]; then
    DEPLOY_CMD="$DEPLOY_CMD --set-env-vars=REDIS_HOST=$REDIS_HOST,REDIS_PORT=$REDIS_PORT"

    if [ "$REDIS_OPTION" = "1" ] && [ -n "$REDIS_PASSWORD" ]; then
        DEPLOY_CMD="$DEPLOY_CMD --set-secrets=REDIS_PASSWORD=redis-password:latest"
    fi
fi

# Add Cloud SQL Unix socket path
DEPLOY_CMD="$DEPLOY_CMD --set-env-vars=DB_HOST=/cloudsql/$DB_CONNECTION_NAME"

# Execute deployment
eval $DEPLOY_CMD

# Get service URL
SERVICE_URL=$(gcloud run services describe shopogoda-bot --region=$REGION --format='get(status.url)')

# Configure Telegram webhook
echo -e "${YELLOW}[8/8] Configuring Telegram webhook...${NC}"
WEBHOOK_RESPONSE=$(curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  -d "url=${SERVICE_URL}/webhook")

if echo "$WEBHOOK_RESPONSE" | grep -q '"ok":true'; then
    echo -e "${GREEN}  Webhook configured successfully!${NC}"
else
    echo -e "${RED}  Warning: Webhook configuration failed. Response: $WEBHOOK_RESPONSE${NC}"
fi

# Display success message
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║        Deployment Completed Successfully!      ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Bot URL:${NC} $SERVICE_URL"
echo -e "${GREEN}Health Check:${NC} ${SERVICE_URL}/health"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Test your bot on Telegram"
echo "2. View logs: gcloud run services logs read shopogoda-bot --region=$REGION"
echo "3. Monitor: https://console.cloud.google.com/run/detail/$REGION/shopogoda-bot"
echo ""
echo -e "${YELLOW}Webhook info:${NC}"
curl -s "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getWebhookInfo" | jq .
echo ""
echo -e "${GREEN}Deployment complete! Your bot should now be responding to messages.${NC}"
