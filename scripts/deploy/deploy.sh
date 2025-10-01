#!/usr/bin/env bash

# ============================================================================
# ShoPogoda Deployment Script
# ============================================================================
#
# This script handles deployment of ShoPogoda to staging or production
# environments with proper health checks, rollback capabilities, and logging.
#
# Usage:
#   ./deploy.sh [staging|production] [version]
#
# Examples:
#   ./deploy.sh staging develop
#   ./deploy.sh production v1.2.3
#   ./deploy.sh production latest
#
# ============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# ============================================================================
# Functions
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_usage() {
    cat << EOF
ShoPogoda Deployment Script

Usage: $0 [ENVIRONMENT] [VERSION]

Arguments:
    ENVIRONMENT     Target environment (staging|production)
    VERSION         Docker image version tag (default: latest for prod, develop for staging)

Examples:
    $0 staging                 # Deploy develop tag to staging
    $0 staging feature-branch  # Deploy feature-branch tag to staging
    $0 production v1.2.3       # Deploy v1.2.3 to production
    $0 production latest       # Deploy latest to production

Environment:
    The script expects .env.{ENVIRONMENT} file in project root
    Example: .env.production or .env.staging

EOF
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi

    # Check if docker-compose is installed
    if ! command -v docker-compose &> /dev/null; then
        log_error "docker-compose is not installed"
        exit 1
    fi

    # Check if env file exists
    if [ ! -f "${PROJECT_ROOT}/.env.${ENVIRONMENT}" ]; then
        log_error "Environment file .env.${ENVIRONMENT} not found"
        log_info "Please create it from .env.${ENVIRONMENT}.example"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

backup_current_deployment() {
    log_info "Creating backup of current deployment..."

    local backup_dir="${PROJECT_ROOT}/backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "${backup_dir}"

    # Save current container state
    docker-compose -f "${COMPOSE_FILE}" ps > "${backup_dir}/containers.txt" 2>&1 || true

    # Save current images
    docker-compose -f "${COMPOSE_FILE}" images > "${backup_dir}/images.txt" 2>&1 || true

    # Save environment file
    cp "${PROJECT_ROOT}/.env.${ENVIRONMENT}" "${backup_dir}/.env.${ENVIRONMENT}.backup" 2>&1 || true

    log_success "Backup created at ${backup_dir}"
    echo "${backup_dir}" > "${PROJECT_ROOT}/.last_backup"
}

pull_latest_images() {
    log_info "Pulling latest Docker images..."

    export VERSION="${VERSION}"
    export GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-valpere/shopogoda}"

    if ! docker-compose -f "${COMPOSE_FILE}" pull bot; then
        log_error "Failed to pull bot image"
        return 1
    fi

    docker-compose -f "${COMPOSE_FILE}" pull postgres redis prometheus grafana jaeger || true

    log_success "Images pulled successfully"
}

start_deployment() {
    log_info "Starting deployment to ${ENVIRONMENT}..."

    export VERSION="${VERSION}"
    export GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-valpere/shopogoda}"

    # Start services
    if ! docker-compose -f "${COMPOSE_FILE}" up -d; then
        log_error "Failed to start services"
        return 1
    fi

    log_success "Services started"
}

wait_for_health() {
    local service=$1
    local max_attempts=${2:-30}
    local attempt=0

    log_info "Waiting for ${service} to become healthy..."

    while [ $attempt -lt $max_attempts ]; do
        if docker-compose -f "${COMPOSE_FILE}" ps | grep "${service}" | grep -q "healthy\|Up"; then
            log_success "${service} is healthy"
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done

    echo ""
    log_error "${service} failed to become healthy"
    return 1
}

verify_deployment() {
    log_info "Verifying deployment..."

    # Wait for database
    wait_for_health "postgres" 30 || return 1

    # Wait for redis
    wait_for_health "redis" 30 || return 1

    # Wait for bot
    wait_for_health "bot" 60 || return 1

    # Test health endpoint
    log_info "Testing bot health endpoint..."
    local bot_port=$(docker-compose -f "${COMPOSE_FILE}" port bot 8080 2>/dev/null | cut -d: -f2)

    if [ -n "${bot_port}" ]; then
        if curl -sf "http://localhost:${bot_port}/health" > /dev/null; then
            log_success "Health endpoint responding correctly"
        else
            log_warning "Health endpoint not responding (this may be normal if webhook mode is used)"
        fi
    fi

    log_success "Deployment verification passed"
}

show_deployment_status() {
    log_info "Deployment Status:"
    echo ""
    docker-compose -f "${COMPOSE_FILE}" ps
    echo ""
    log_info "Logs: docker-compose -f ${COMPOSE_FILE} logs -f"
}

rollback() {
    log_warning "Rolling back deployment..."

    if [ ! -f "${PROJECT_ROOT}/.last_backup" ]; then
        log_error "No backup found for rollback"
        return 1
    fi

    local backup_dir=$(cat "${PROJECT_ROOT}/.last_backup")

    # Stop current deployment
    docker-compose -f "${COMPOSE_FILE}" down

    # Restore backup environment
    if [ -f "${backup_dir}/.env.${ENVIRONMENT}.backup" ]; then
        cp "${backup_dir}/.env.${ENVIRONMENT}.backup" "${PROJECT_ROOT}/.env.${ENVIRONMENT}"
    fi

    # Restart with previous version
    docker-compose -f "${COMPOSE_FILE}" up -d

    log_success "Rollback completed"
}

cleanup_old_backups() {
    log_info "Cleaning up old backups (keeping last 5)..."

    local backup_base="${PROJECT_ROOT}/backups"
    if [ -d "${backup_base}" ]; then
        find "${backup_base}" -maxdepth 1 -type d | sort -r | tail -n +6 | xargs rm -rf 2>/dev/null || true
    fi

    log_success "Old backups cleaned"
}

# ============================================================================
# Main Script
# ============================================================================

# Parse arguments
if [ $# -lt 1 ]; then
    show_usage
    exit 1
fi

ENVIRONMENT=$1
VERSION=${2:-}

# Set default version based on environment
if [ -z "${VERSION}" ]; then
    if [ "${ENVIRONMENT}" = "production" ]; then
        VERSION="latest"
    else
        VERSION="develop"
    fi
fi

# Validate environment
if [ "${ENVIRONMENT}" != "staging" ] && [ "${ENVIRONMENT}" != "production" ]; then
    log_error "Invalid environment: ${ENVIRONMENT}"
    show_usage
    exit 1
fi

# Set compose file
COMPOSE_FILE="${PROJECT_ROOT}/docker/docker-compose.${ENVIRONMENT}.yml"

if [ ! -f "${COMPOSE_FILE}" ]; then
    log_error "Compose file not found: ${COMPOSE_FILE}"
    exit 1
fi

# Main deployment flow
log_info "========================================="
log_info "ShoPogoda Deployment"
log_info "========================================="
log_info "Environment: ${ENVIRONMENT}"
log_info "Version: ${VERSION}"
log_info "Compose File: ${COMPOSE_FILE}"
log_info "========================================="
echo ""

# Execute deployment steps
check_prerequisites
backup_current_deployment
pull_latest_images

if start_deployment && verify_deployment; then
    show_deployment_status
    cleanup_old_backups
    log_success "========================================="
    log_success "Deployment to ${ENVIRONMENT} completed successfully!"
    log_success "========================================="
    exit 0
else
    log_error "Deployment failed!"
    log_warning "Run './deploy.sh rollback ${ENVIRONMENT}' to rollback"
    exit 1
fi
