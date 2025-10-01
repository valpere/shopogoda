# Code Quality & Coverage Standards

This document outlines the code quality standards and coverage requirements for the ShoPogoda project.

## Coverage Requirements

### Current Status
- **Current Coverage**: ~28.2%
- **Minimum Threshold (main branch)**: 25%
- **Minimum Threshold (PRs)**: 20%
- **Aspirational Target**: 60%

### Coverage Thresholds by Branch

| Branch | Minimum Coverage | Status |
|--------|-----------------|--------|
| `main` | 25% | ✅ Enforced in CI |
| `develop` | 20% | ✅ Enforced in CI |
| PRs | 20% | ✅ Enforced in PR checks |

### Coverage Improvement Plan

We're progressively increasing coverage requirements:

1. **Phase 1** (Current): 25% on main, 20% on PRs
2. **Phase 2** (Target Q2 2025): 40% on main, 35% on PRs
3. **Phase 3** (Target Q4 2025): 60% on main, 55% on PRs

## Code Quality Checks

### 1. Cyclomatic Complexity

**Tool**: `gocyclo`
**Threshold**: Functions with complexity > 15 trigger warnings

Cyclomatic complexity measures the number of independent paths through code. High complexity indicates:
- Functions that are hard to test
- Increased likelihood of bugs
- Difficulty in maintenance

**How to check locally:**
```bash
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
gocyclo -over 15 .
```

**Best practices:**
- Keep functions focused on a single responsibility
- Extract complex logic into smaller helper functions
- Aim for complexity < 10 for most functions

### 2. Code Duplication

**Tool**: `dupl`
**Threshold**: Blocks > 100 tokens

Code duplication leads to:
- Increased maintenance burden
- Inconsistent bug fixes
- Larger codebase

**How to check locally:**
```bash
go install github.com/mibk/dupl@latest
dupl -threshold 100 ./...
```

**Best practices:**
- Extract common logic into shared functions
- Use interfaces for polymorphic behavior
- Consider design patterns (Strategy, Template Method)

### 3. Performance Benchmarks

**Tool**: `benchstat`
**Regression Threshold**: >10% performance degradation triggers warnings

Continuous performance monitoring ensures:
- No unintended performance regressions
- Optimization efforts are measurable
- Performance-critical code is tracked

**How to run benchmarks locally:**
```bash
go test -bench=. -benchmem -benchtime=3s ./...
```

**Benchmark comparison:**
```bash
# Save baseline
go test -bench=. -benchmem ./... > old.txt

# Make changes, then compare
go test -bench=. -benchmem ./... > new.txt
benchstat old.txt new.txt
```

### 4. Static Analysis

**Tools**:
- `golangci-lint`: Comprehensive linter suite
- `staticcheck`: Advanced static analysis
- `go vet`: Standard Go analysis

**How to run locally:**
```bash
# Quick check
make lint

# Comprehensive check
golangci-lint run --timeout=5m

# Static analysis
staticcheck ./...
```

### 5. Security Scanning

**Tools**:
- `gosec`: Security-focused Go analyzer
- `govulncheck`: Go vulnerability database checker
- `trivy`: Container and filesystem vulnerability scanner

**How to run locally:**
```bash
# Security scan
gosec ./...

# Vulnerability check
govulncheck ./...

# Docker image scan
trivy image shopogoda:latest
```

## Automated Quality Gates

### Pull Request Checks

Every PR automatically runs:
1. ✅ Code formatting check (`gofmt`)
2. ✅ Go vet
3. ✅ Linter (only new issues)
4. ✅ Cyclomatic complexity check
5. ✅ Code duplication detection
6. ✅ Unit tests with race detection
7. ✅ Coverage check (≥20%)
8. ✅ Security scan
9. ✅ Conventional commit message format
10. ✅ Breaking change detection

### Main Branch Checks

Merges to `main` additionally run:
1. ✅ Full test suite (unit + integration)
2. ✅ Performance benchmarks with regression detection
3. ✅ Docker image build and security scan
4. ✅ CodeQL analysis
5. ✅ Coverage check (≥25%)

### Weekly Code Quality Analysis

Every Monday at 3 AM UTC, a comprehensive code quality report is generated:
- Code complexity trends
- Duplication analysis
- Spelling checks
- Security vulnerabilities
- Code metrics (LOC, test coverage)

**View reports**: Actions → Code Quality Analysis → Artifacts

## Dependency Management

**Dependabot** is configured to automatically:
- Check for Go module updates (weekly)
- Check for GitHub Actions updates (weekly)
- Check for Docker base image updates (weekly)
- Create PRs with dependency updates
- Label PRs appropriately

**Configuration**: `.github/dependabot.yml`

## Local Development Workflow

### Before Committing

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests with coverage
make test-coverage

# Check security
gosec ./...
govulncheck ./...
```

### Pre-Push Checklist

- [ ] All tests pass (`make test`)
- [ ] Coverage meets minimum threshold
- [ ] No linter errors
- [ ] No security issues
- [ ] Commit messages follow conventional format
- [ ] No high-complexity functions added

### Running Full CI Locally

```bash
make ci-local
```

This runs the complete CI pipeline locally before pushing.

## Quality Metrics Dashboard

### Current Metrics (Updated: 2025-10-01)

| Metric | Value | Target | Status |
|--------|-------|--------|--------|
| Test Coverage | 28.2% | 60% | 🟡 Improving |
| Packages with Tests | 8/19 | 19/19 | 🟡 Partial |
| Average Complexity | ~8 | <10 | ✅ Good |
| Security Issues | 0 | 0 | ✅ Clean |
| Code Duplication | Low | Minimal | ✅ Good |

### Package-Level Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| `internal/config` | 98.2% | ✅ Excellent |
| `internal/models` | 99.0% | ✅ Excellent |
| `internal/middleware` | 97.2% | ✅ Excellent |
| `pkg/weather` | 97.6% | ✅ Excellent |
| `pkg/metrics` | 100% | ✅ Perfect |
| `pkg/alerts` | 99.2% | ✅ Excellent |
| `internal/services` | 80.9% | ✅ Good |
| `internal/database` | 50.0% | 🟡 Needs Work |
| `internal/bot` | 0% | 🔴 Critical |
| `internal/handlers/commands` | 0% | 🔴 Critical |

**Priority**: Focus on `internal/bot` and `internal/handlers/commands` packages.

## Contributing to Quality Improvements

### Adding Tests

When adding tests:
1. Test both success and error paths
2. Use table-driven tests for multiple scenarios
3. Mock external dependencies
4. Aim for 80%+ coverage for new code
5. Include edge cases and boundary conditions

### Refactoring for Quality

When refactoring:
1. Extract complex functions into smaller ones
2. Reduce cyclomatic complexity
3. Eliminate code duplication
4. Add tests before refactoring
5. Verify benchmarks don't regress

### Reviewing PRs

Quality checklist for reviewers:
- [ ] Tests added for new functionality
- [ ] Coverage hasn't decreased
- [ ] No new high-complexity functions
- [ ] No security warnings introduced
- [ ] Performance benchmarks reviewed (if applicable)
- [ ] Documentation updated

## Tools Installation

Install all quality tools locally:

```bash
make install-tools
```

This installs:
- golangci-lint
- goimports
- mockgen
- govulncheck
- gocyclo
- dupl
- staticcheck
- gosec
- benchstat

## Continuous Improvement

We continuously improve code quality through:
1. **Weekly Reviews**: Code quality reports analyzed
2. **Monthly Goals**: Incremental coverage targets
3. **Automated Updates**: Dependabot keeps dependencies current
4. **Team Education**: Sharing best practices
5. **Tool Upgrades**: Staying current with analysis tools

## Resources

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [Codecov Coverage Guide](https://docs.codecov.com/docs)

## Questions?

For questions about code quality standards, open an issue with the `quality` label.
