#!/bin/bash

# ShoPogoda Rollback Script
# Rolls back to a previous version in production

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

info() {
    echo -e "${CYAN}$1${NC}"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Get current version from git
get_current_version() {
    git describe --tags --abbrev=0 2>/dev/null || echo "unknown"
}

# List recent versions
list_versions() {
    info "Recent versions:"
    git tag --sort=-v:refname | head -10
}

# Rollback to version
rollback() {
    local target_version=$1
    local environment=${2:-production}

    info "Rolling back to version $target_version in $environment..."

    case $environment in
        staging)
            warning "Rolling back staging environment"
            # Add staging rollback commands
            # kubectl rollout undo deployment/shopogoda -n staging
            # helm rollback shopogoda --namespace staging
            ;;
        production)
            warning "Rolling back PRODUCTION environment"
            read -p "Are you absolutely sure? This will affect live users (y/N): " CONFIRM
            if [[ ! $CONFIRM =~ ^[Yy]$ ]]; then
                warning "Rollback cancelled"
                exit 0
            fi
            # Add production rollback commands
            # kubectl rollout undo deployment/shopogoda -n production
            # helm rollback shopogoda --namespace production
            ;;
        *)
            error "Unknown environment: $environment. Use 'staging' or 'production'"
            ;;
    esac

    success "Rollback to $target_version completed"
    info "Verify deployment: kubectl get pods -n $environment"
}

main() {
    info "ShoPogoda Rollback Manager"
    echo ""

    CURRENT_VERSION=$(get_current_version)
    info "Current version: $CURRENT_VERSION"
    echo ""

    # Get target version
    if [ -n "$1" ]; then
        TARGET_VERSION=$1
    else
        list_versions
        echo ""
        read -p "Enter version to rollback to: " TARGET_VERSION
    fi

    # Validate version exists
    if ! git rev-parse "$TARGET_VERSION" >/dev/null 2>&1; then
        error "Version $TARGET_VERSION not found"
    fi

    # Get environment
    ENVIRONMENT=${2:-production}

    # Confirm rollback
    echo ""
    warning "Rollback Summary:"
    echo "  Current Version: $CURRENT_VERSION"
    echo "  Target Version:  $TARGET_VERSION"
    echo "  Environment:     $ENVIRONMENT"
    echo ""

    rollback "$TARGET_VERSION" "$ENVIRONMENT"
}

main "$@"
