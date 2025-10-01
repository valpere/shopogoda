#!/usr/bin/env bash

# ============================================================================
# ShoPogoda Rollback Script
# ============================================================================
#
# This script handles rollback of ShoPogoda deployments
#
# Usage:
#   ./rollback.sh [staging|production]
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
ShoPogoda Rollback Script

Usage: $0 [ENVIRONMENT]

Arguments:
    ENVIRONMENT     Target environment (staging|production)

Examples:
    $0 staging      # Rollback staging deployment
    $0 production   # Rollback production deployment

EOF
}

list_backups() {
    local backup_base="${PROJECT_ROOT}/backups"

    if [ ! -d "${backup_base}" ]; then
        log_warning "No backups directory found"
        return 1
    fi

    log_info "Available backups:"
    echo ""
    local count=1
    for backup in $(find "${backup_base}" -maxdepth 1 -type d | sort -r | tail -n +2); do
        echo "${count}. $(basename ${backup})"
        count=$((count + 1))
    done
    echo ""
}

select_backup() {
    list_backups

    echo -n "Select backup number (or press Enter for latest): "
    read selection

    local backup_base="${PROJECT_ROOT}/backups"
    if [ -z "${selection}" ]; then
        # Use latest
        BACKUP_DIR=$(find "${backup_base}" -maxdepth 1 -type d | sort -r | head -n 2 | tail -n 1)
    else
        # Use selected
        BACKUP_DIR=$(find "${backup_base}" -maxdepth 1 -type d | sort -r | tail -n +2 | sed -n "${selection}p")
    fi

    if [ -z "${BACKUP_DIR}" ] || [ ! -d "${BACKUP_DIR}" ]; then
        log_error "Invalid backup selection"
        return 1
    fi

    log_info "Selected backup: $(basename ${BACKUP_DIR})"
}

confirm_rollback() {
    log_warning "========================================="
    log_warning "ROLLBACK CONFIRMATION"
    log_warning "========================================="
    log_warning "Environment: ${ENVIRONMENT}"
    log_warning "Backup: $(basename ${BACKUP_DIR})"
    log_warning "========================================="
    echo ""
    echo -n "Are you sure you want to rollback? (yes/no): "
    read confirmation

    if [ "${confirmation}" != "yes" ]; then
        log_info "Rollback cancelled"
        exit 0
    fi
}

perform_rollback() {
    log_info "Performing rollback..."

    # Stop current deployment
    log_info "Stopping current deployment..."
    docker-compose -f "${COMPOSE_FILE}" down

    # Restore environment file if exists
    if [ -f "${BACKUP_DIR}/.env.${ENVIRONMENT}.backup" ]; then
        log_info "Restoring environment configuration..."
        cp "${BACKUP_DIR}/.env.${ENVIRONMENT}.backup" "${PROJECT_ROOT}/.env.${ENVIRONMENT}"
    fi

    # Extract version from backup if available
    if [ -f "${BACKUP_DIR}/images.txt" ]; then
        log_info "Extracting version from backup..."
        # This is a simplified version extraction
        # In production, you might want to parse the actual image tags
    fi

    # Start with previous configuration
    log_info "Starting services..."
    if docker-compose -f "${COMPOSE_FILE}" up -d; then
        log_success "Services started successfully"
    else
        log_error "Failed to start services"
        return 1
    fi

    # Wait for services
    sleep 10

    # Verify services are running
    log_info "Verifying services..."
    docker-compose -f "${COMPOSE_FILE}" ps

    log_success "Rollback completed successfully"
}

# ============================================================================
# Main Script
# ============================================================================

if [ $# -lt 1 ]; then
    show_usage
    exit 1
fi

ENVIRONMENT=$1

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

# Execute rollback
select_backup
confirm_rollback
perform_rollback

log_success "========================================="
log_success "Rollback to ${ENVIRONMENT} completed!"
log_success "========================================="
