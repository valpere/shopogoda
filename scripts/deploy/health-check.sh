#!/usr/bin/env bash

# ============================================================================
# ShoPogoda Health Check Script
# ============================================================================
#
# Comprehensive health check for all ShoPogoda services
#
# Usage:
#   ./health-check.sh [staging|production]
#
# ============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

check_service_health() {
    local service=$1
    local container_name=$2

    echo -n "Checking ${service}... "

    if ! docker ps --filter "name=${container_name}" --format "{{.Names}}" | grep -q "${container_name}"; then
        log_error "${service} container not running"
        return 1
    fi

    local health=$(docker inspect --format='{{.State.Health.Status}}' "${container_name}" 2>/dev/null || echo "none")

    case "${health}" in
        "healthy")
            log_success "${service} is healthy"
            return 0
            ;;
        "starting")
            log_warning "${service} is starting"
            return 1
            ;;
        "unhealthy")
            log_error "${service} is unhealthy"
            return 1
            ;;
        "none")
            # No health check defined, check if container is running
            local state=$(docker inspect --format='{{.State.Status}}' "${container_name}" 2>/dev/null)
            if [ "${state}" = "running" ]; then
                log_success "${service} is running (no health check defined)"
                return 0
            else
                log_error "${service} is not running"
                return 1
            fi
            ;;
        *)
            log_warning "${service} health status unknown: ${health}"
            return 1
            ;;
    esac
}

check_http_endpoint() {
    local service=$1
    local url=$2

    echo -n "Checking ${service} HTTP endpoint... "

    if curl -sf "${url}" > /dev/null 2>&1; then
        log_success "${service} endpoint responding"
        return 0
    else
        log_error "${service} endpoint not responding"
        return 1
    fi
}

check_database_connection() {
    local container_name=$1

    echo -n "Checking database connection... "

    if docker exec "${container_name}" pg_isready -U shopogoda > /dev/null 2>&1; then
        log_success "Database connection OK"
        return 0
    else
        log_error "Database connection failed"
        return 1
    fi
}

check_redis_connection() {
    local container_name=$1

    echo -n "Checking Redis connection... "

    if docker exec "${container_name}" redis-cli ping | grep -q "PONG"; then
        log_success "Redis connection OK"
        return 0
    else
        log_error "Redis connection failed"
        return 1
    fi
}

show_resource_usage() {
    echo ""
    log_info "Resource Usage:"
    echo ""
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" \
        $(docker-compose -f "${COMPOSE_FILE}" ps -q 2>/dev/null)
}

show_service_logs() {
    local service=$1
    local lines=${2:-50}

    echo ""
    log_info "Last ${lines} lines from ${service}:"
    echo ""
    docker-compose -f "${COMPOSE_FILE}" logs --tail="${lines}" "${service}"
}

# Main
if [ $# -lt 1 ]; then
    echo "Usage: $0 [staging|production]"
    exit 1
fi

ENVIRONMENT=$1
COMPOSE_FILE="${PROJECT_ROOT}/docker/docker-compose.${ENVIRONMENT}.yml"

if [ ! -f "${COMPOSE_FILE}" ]; then
    log_error "Compose file not found: ${COMPOSE_FILE}"
    exit 1
fi

# Determine container name prefix based on environment
PREFIX="shopogoda"
if [ "${ENVIRONMENT}" = "production" ]; then
    SUFFIX="prod"
else
    SUFFIX="staging"
fi

log_info "========================================="
log_info "ShoPogoda Health Check - ${ENVIRONMENT}"
log_info "========================================="
echo ""

all_healthy=0

# Check all services
check_service_health "PostgreSQL" "${PREFIX}-db-${SUFFIX}" || all_healthy=1
check_service_health "Redis" "${PREFIX}-redis-${SUFFIX}" || all_healthy=1
check_service_health "Bot" "${PREFIX}-bot-${SUFFIX}" || all_healthy=1
check_service_health "Prometheus" "${PREFIX}-prometheus-${SUFFIX}" || all_healthy=1
check_service_health "Grafana" "${PREFIX}-grafana-${SUFFIX}" || all_healthy=1

echo ""

# Additional connection checks
check_database_connection "${PREFIX}-db-${SUFFIX}" || all_healthy=1
check_redis_connection "${PREFIX}-redis-${SUFFIX}" || all_healthy=1

# HTTP endpoint checks
if [ "${ENVIRONMENT}" = "production" ]; then
    BOT_PORT=8080
    GRAFANA_PORT=3000
    PROMETHEUS_PORT=9090
else
    BOT_PORT=8081
    GRAFANA_PORT=3001
    PROMETHEUS_PORT=9091
fi

echo ""
check_http_endpoint "Bot" "http://localhost:${BOT_PORT}/health" || all_healthy=1
check_http_endpoint "Grafana" "http://localhost:${GRAFANA_PORT}/api/health" || all_healthy=1
check_http_endpoint "Prometheus" "http://localhost:${PROMETHEUS_PORT}/-/healthy" || all_healthy=1

# Show resource usage
show_resource_usage

echo ""
log_info "========================================="
if [ $all_healthy -eq 0 ]; then
    log_success "All services are healthy!"
    log_info "========================================="
    exit 0
else
    log_error "Some services are unhealthy!"
    log_info "========================================="
    echo ""
    echo "Run: docker-compose -f ${COMPOSE_FILE} logs [service_name]"
    echo "for more details"
    exit 1
fi
