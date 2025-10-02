# Release Process

This document describes the release process for ShoPogoda.

## Semantic Versioning

ShoPogoda follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH[-PRERELEASE]
```

- **MAJOR**: Breaking changes, major architectural changes
- **MINOR**: New features, backwards-compatible
- **PATCH**: Bug fixes, security patches
- **PRERELEASE**: `-demo`, `-beta.1`, `-rc.1`, etc.

### Examples

- `v0.1.0-demo` - Demo release
- `v0.2.0-beta.1` - Beta release
- `v1.0.0-rc.1` - Release candidate
- `v1.0.0` - Stable release
- `v1.0.1` - Patch release
- `v1.1.0` - Minor feature release
- `v2.0.0` - Major release with breaking changes

## Release Types

### Development Releases (`-dev`, `-alpha`)
- **Purpose**: Internal testing, experimental features
- **Frequency**: As needed
- **Support**: None
- **Deployment**: Local/dev environments only

### Demo Releases (`-demo`)
- **Purpose**: Public demos, portfolio showcases
- **Frequency**: Major milestones
- **Support**: Limited (best effort)
- **Deployment**: Demo environments, documentation

### Beta Releases (`-beta.X`)
- **Purpose**: Feature testing, community feedback
- **Frequency**: Before minor releases
- **Support**: Bug fixes only
- **Deployment**: Staging environments

### Release Candidates (`-rc.X`)
- **Purpose**: Final testing before stable release
- **Frequency**: Before major/minor releases
- **Support**: Bug fixes and security patches
- **Deployment**: Staging and pre-production

### Stable Releases (no suffix)
- **Purpose**: Production deployments
- **Frequency**: Every 2-3 months (minor), 6-9 months (major)
- **Support**: Full support (features + bug fixes)
- **Deployment**: Production environments

## Creating a Release

### Prerequisites

1. **Clean working directory**
   ```bash
   git status
   # Ensure no uncommitted changes
   ```

2. **All tests passing**
   ```bash
   make test
   make lint
   ```

3. **Updated documentation**
   - CHANGELOG.md has unreleased changes documented
   - README.md is up to date
   - Any new features documented

4. **Code coverage meets requirements**
   ```bash
   make test-coverage
   # Check coverage.html
   ```

### Automated Release (Recommended)

Use the release creation script:

```bash
./scripts/create-release.sh v0.1.0-demo
```

This script will:
1. Validate version format
2. Check git is clean
3. Update CHANGELOG.md
4. Commit changelog changes
5. Create git tag
6. Provide push instructions

### Manual Release

If you prefer manual control:

1. **Update CHANGELOG.md**
   ```bash
   # Move [Unreleased] section content to new version section
   # Add new empty [Unreleased] section
   ```

2. **Commit changelog**
   ```bash
   git add CHANGELOG.md
   git commit -m "docs: Update CHANGELOG for v0.1.0-demo"
   ```

3. **Create tag**
   ```bash
   git tag -a v0.1.0-demo -m "Release v0.1.0-demo"
   ```

4. **Push tag**
   ```bash
   git push origin v0.1.0-demo
   git push origin main  # Don't forget the changelog commit
   ```

### What Happens Next

When you push a tag matching `v*.*.*`, the GitHub Actions release workflow automatically:

1. **Runs all tests** to ensure quality
2. **Builds binaries** for multiple platforms:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)
   - Windows (amd64)
3. **Builds Docker images** with version tags
4. **Pushes to GitHub Container Registry**
5. **Creates GitHub Release** with:
   - Changelog
   - Binary downloads
   - Docker pull commands
   - SHA256 checksums
6. **Deploys** based on version:
   - Pre-releases (`-demo`, `-beta`, `-rc`) → Staging
   - Stable releases → Production

## Release Checklist

Before creating a release, verify:

- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Test coverage meets minimum (40% for demo, 60% for stable)
- [ ] CHANGELOG.md updated with all changes
- [ ] README.md reflects current features
- [ ] Documentation is up to date
- [ ] No known critical bugs
- [ ] Security vulnerabilities addressed
- [ ] Dependencies up to date
- [ ] `.env.example` has all required variables
- [ ] Database migrations tested
- [ ] Backwards compatibility verified (or breaking changes documented)

### For Stable Releases (v1.0.0+)

Additional requirements:

- [ ] E2E tests pass
- [ ] Performance benchmarks meet targets
- [ ] Security audit completed
- [ ] Load testing passed
- [ ] Monitoring dashboards configured
- [ ] Rollback plan documented
- [ ] Release notes reviewed
- [ ] Beta testing completed (if applicable)

## Deployment

### Staging Deployment

Pre-releases automatically deploy to staging:

```bash
# Triggered by pushing tags like:
git push origin v0.2.0-beta.1
```

### Production Deployment

Stable releases automatically deploy to production:

```bash
# Triggered by pushing tags like:
git push origin v1.0.0
```

### Manual Deployment

If you need to deploy manually:

```bash
# Pull latest Docker image
docker pull ghcr.io/valpere/shopogoda:0.1.0-demo

# Or deploy with Kubernetes
kubectl set image deployment/shopogoda \
  shopogoda=ghcr.io/valpere/shopogoda:0.1.0-demo \
  -n production
```

## Rollback Procedure

If a release has critical issues:

### Automated Rollback

```bash
./scripts/rollback.sh v0.0.9 production
```

### Manual Rollback

```bash
# Kubernetes
kubectl rollout undo deployment/shopogoda -n production

# Docker Compose
docker compose pull  # Pull previous version
docker compose up -d --no-deps shopogoda

# Verify rollback
kubectl get pods -n production
docker compose ps
```

## Hotfix Process

For critical bugs in production:

1. **Create hotfix branch from production tag**
   ```bash
   git checkout -b hotfix/v1.0.1 v1.0.0
   ```

2. **Fix the bug**
   ```bash
   # Make minimal changes to fix critical issue
   git commit -m "fix: Critical bug description"
   ```

3. **Test thoroughly**
   ```bash
   make test
   make test-integration
   ```

4. **Create hotfix release**
   ```bash
   ./scripts/create-release.sh v1.0.1
   ```

5. **Merge back to main**
   ```bash
   git checkout main
   git merge hotfix/v1.0.1
   git push origin main
   ```

## Version Verification

### Check Version in Running Bot

```bash
# Using Telegram bot
/version

# Using API
curl http://localhost:8080/health
```

### Check Version in Docker Image

```bash
docker run --rm ghcr.io/valpere/shopogoda:latest /app/shopogoda --version
```

### Check Version in Binary

```bash
./shopogoda --version
```

## Release Cadence

### Regular Schedule

- **Patch releases**: As needed (bug fixes, security)
- **Minor releases**: Every 2-3 months (new features)
- **Major releases**: Every 6-9 months (breaking changes)

### Special Releases

- **Demo releases**: Major milestones (v0.1.0-demo, v0.2.0-demo)
- **Beta releases**: 2-4 weeks before minor/major releases
- **Release candidates**: 1 week before major releases

## Support Policy

### Active Support

- **Latest major version**: Full support (features + bug fixes)
- **Previous major version**: Security patches only (12 months)
- **Older versions**: End of life (no support)

### Example

If current version is v2.1.0:
- **v2.x.x**: Full support
- **v1.x.x**: Security patches only (until v2.0.0 + 12 months)
- **v0.x.x**: End of life

## Troubleshooting

### Release workflow failed

**Check GitHub Actions logs:**
```bash
gh run list --workflow=release.yml --limit 5
gh run view <run-id> --log-failed
```

**Common issues:**
- Test failures → Fix tests before releasing
- Docker build failures → Check Dockerfile syntax
- Permission errors → Verify GITHUB_TOKEN has correct scopes

### Tag already exists

**If you need to recreate a tag:**
```bash
# Delete local tag
git tag -d v0.1.0-demo

# Delete remote tag (CAUTION: only for unreleased tags)
git push origin :refs/tags/v0.1.0-demo

# Recreate tag
git tag -a v0.1.0-demo -m "Release v0.1.0-demo"
git push origin v0.1.0-demo
```

### Rollback needed immediately

**Emergency rollback:**
```bash
# Production
./scripts/rollback.sh v1.0.0 production

# Verify
kubectl get pods -n production -w
```

## Metrics & Monitoring

### Post-Release Monitoring

After each release, monitor:

1. **Error rates** (should not increase)
   - Check Grafana dashboards
   - Review error logs

2. **Response times** (should remain <200ms)
   - Prometheus metrics
   - Application performance

3. **User reports** (check for new issues)
   - GitHub Issues
   - Support channels

4. **Resource usage** (CPU, memory)
   - Kubernetes metrics
   - Container stats

### Release Success Criteria

A release is considered successful if:

- No increase in error rate (≤1% of requests)
- Response times within SLA (<200ms p95)
- No critical bugs reported in first 24 hours
- Monitoring shows healthy metrics
- Rollback not needed within 72 hours

## Contact

For release management questions:
- **Project Lead**: Valentyn Solomko (valentyn.solomko@gmail.com)
- **GitHub Issues**: https://github.com/valpere/shopogoda/issues

---

**Last Updated**: 2025-01-02
