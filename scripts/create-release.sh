#!/bin/bash

# ShoPogoda Release Creation Script
# Creates a new release with proper versioning and changelog

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Helper functions
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

# Check if git is clean
check_git_clean() {
    if ! git diff-index --quiet HEAD --; then
        error "Working directory has uncommitted changes. Commit or stash them first."
    fi
    success "Git working directory is clean"
}

# Validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-z0-9]+(\.[0-9]+)?)?$ ]]; then
        error "Invalid version format. Use semantic versioning (e.g., v1.0.0, v0.1.0-demo, v1.2.3-beta.1)"
    fi
    success "Version format is valid: $version"
}

# Check if version already exists
check_version_exists() {
    local version=$1
    if git rev-parse "$version" >/dev/null 2>&1; then
        error "Version $version already exists"
    fi
    success "Version $version is available"
}

# Update CHANGELOG.md
update_changelog() {
    local version=$1
    local version_no_v=${version#v}
    local date=$(date +%Y-%m-%d)

    info "Updating CHANGELOG.md..."

    # Check if CHANGELOG.md exists
    if [ ! -f CHANGELOG.md ]; then
        error "CHANGELOG.md not found"
    fi

    # Check if version already exists in changelog
    if grep -q "## \[$version_no_v\]" CHANGELOG.md; then
        warning "Version $version_no_v already in CHANGELOG.md, skipping update"
        return
    fi

    # Move [Unreleased] content to new version
    if grep -q "## \[Unreleased\]" CHANGELOG.md; then
        # Create temp file with updated changelog
        awk -v version="$version_no_v" -v date="$date" '
            /^## \[Unreleased\]/ {
                print
                print ""
                print "### In Progress"
                print "- TBD"
                print ""
                print "---"
                print ""
                print "## [" version "] - " date
                in_unreleased = 1
                next
            }
            /^## \[/ && in_unreleased {
                in_unreleased = 0
            }
            !in_unreleased || !/^### In Progress/ {
                print
            }
        ' CHANGELOG.md > CHANGELOG.md.tmp

        mv CHANGELOG.md.tmp CHANGELOG.md
        success "Updated CHANGELOG.md with version $version_no_v"
    else
        warning "No [Unreleased] section found in CHANGELOG.md"
    fi
}

# Create git tag
create_tag() {
    local version=$1
    local message=$2

    info "Creating git tag $version..."
    git tag -a "$version" -m "$message"
    success "Tag $version created"
}

# Main script
main() {
    info "ShoPogoda Release Creator"
    echo ""

    # Get version from argument or prompt
    if [ -n "$1" ]; then
        VERSION=$1
    else
        read -p "Enter version (e.g., v0.1.0-demo): " VERSION
    fi

    # Validate inputs
    validate_version "$VERSION"
    check_git_clean
    check_version_exists "$VERSION"

    # Get release message
    read -p "Enter release message (default: 'Release $VERSION'): " RELEASE_MESSAGE
    RELEASE_MESSAGE=${RELEASE_MESSAGE:-"Release $VERSION"}

    # Confirm
    echo ""
    info "Release Summary:"
    echo "  Version: $VERSION"
    echo "  Message: $RELEASE_MESSAGE"
    echo ""
    read -p "Proceed with release? (y/N): " CONFIRM

    if [[ ! $CONFIRM =~ ^[Yy]$ ]]; then
        warning "Release cancelled"
        exit 0
    fi

    # Update changelog
    update_changelog "$VERSION"

    # Commit changelog update
    if git diff --quiet CHANGELOG.md; then
        info "No changelog changes to commit"
    else
        info "Committing changelog update..."
        git add CHANGELOG.md
        git commit -m "docs: Update CHANGELOG for $VERSION"
        success "Changelog committed"
    fi

    # Create tag
    create_tag "$VERSION" "$RELEASE_MESSAGE"

    # Instructions
    echo ""
    success "Release preparation complete!"
    echo ""
    info "Next steps:"
    echo "  1. Review the changes:"
    echo "     git show $VERSION"
    echo ""
    echo "  2. Push the tag to trigger release workflow:"
    echo "     git push origin $VERSION"
    echo ""
    echo "  3. The GitHub Actions workflow will:"
    echo "     - Run all tests"
    echo "     - Build binaries for all platforms"
    echo "     - Build and push Docker images"
    echo "     - Create GitHub release with assets"
    echo ""
    warning "Remember to also push the changelog commit:"
    echo "     git push origin main"
}

# Run main
main "$@"
