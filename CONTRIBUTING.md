# Contributing to ShoPogoda

Thank you for your interest in contributing to ShoPogoda! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Quality Standards](#code-quality-standards)
- [Testing Requirements](#testing-requirements)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)
- [Documentation](#documentation)
- [Community](#community)

---

## Code of Conduct

ShoPogoda follows a welcoming and inclusive Code of Conduct. Please be respectful and constructive in all interactions.

### Our Standards

- **Be Respectful**: Treat everyone with respect and consideration
- **Be Collaborative**: Work together and help each other
- **Be Professional**: Maintain professionalism in all interactions
- **Be Inclusive**: Welcome contributors from all backgrounds

---

## Getting Started

### Prerequisites

- **Go 1.24+**: [Installation guide](https://golang.org/doc/install)
- **Docker & Docker Compose**: [Installation guide](https://docs.docker.com/get-docker/)
- **Git**: Version control
- **Make**: Build automation

### Initial Setup

1. **Fork the Repository**
   ```bash
   # Click "Fork" on GitHub, then clone your fork
   git clone https://github.com/YOUR_USERNAME/shopogoda.git
   cd shopogoda
   ```

2. **Add Upstream Remote**
   ```bash
   git remote add upstream https://github.com/valpere/shopogoda.git
   git fetch upstream
   ```

3. **Initialize Development Environment**
   ```bash
   make init
   ```

4. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your API keys:
   # - TELEGRAM_BOT_TOKEN (from @BotFather)
   # - OPENWEATHER_API_KEY (from openweathermap.org)
   ```

5. **Start Development Services**
   ```bash
   make dev
   ```

6. **Verify Setup**
   ```bash
   make test
   curl http://localhost:8080/health
   ```

### Development Tools

Install all required development tools:

```bash
make install-tools
```

This installs:
- golangci-lint (linting)
- goimports (import formatting)
- mockgen (test mocking)
- govulncheck (security)
- gocyclo (complexity analysis)
- dupl (duplication detection)
- staticcheck (static analysis)
- gosec (security scanning)
- benchstat (benchmark comparison)

---

## Development Workflow

### 1. Create a Feature Branch

```bash
# Update your fork
git checkout main
git pull upstream main

# Create feature branch
git checkout -b feature/your-feature-name
```

Branch naming conventions:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `test/` - Test improvements
- `refactor/` - Code refactoring
- `perf/` - Performance improvements

### 2. Make Your Changes

- Write clean, readable code
- Follow Go best practices
- Add tests for new functionality
- Update documentation as needed

### 3. Run Quality Checks

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Check test coverage
make test-coverage

# Security scan
gosec ./...
govulncheck ./...
```

### 4. Commit Your Changes

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
git add .
git commit -m "feat: add weather alert customization"
```

See [Commit Message Guidelines](#commit-message-guidelines) below for details.

### 5. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

---

## Code Quality Standards

### Coverage Requirements

| Branch | Minimum Coverage | Status |
|--------|-----------------|--------|
| `main` | 25% | ✅ Enforced in CI |
| PRs | 20% | ✅ Enforced in PR checks |

**Aspirational Target**: 60%+ coverage

### Quality Checks

1. **Cyclomatic Complexity**: Functions with complexity > 15 trigger warnings
2. **Code Duplication**: Blocks > 100 tokens flagged
3. **Security**: No `gosec` or `govulncheck` issues
4. **Linting**: No `golangci-lint` errors
5. **Formatting**: Code must pass `gofmt` check

### Before Committing Checklist

- [ ] All tests pass (`make test`)
- [ ] Coverage meets minimum threshold
- [ ] No linter errors (`make lint`)
- [ ] No security issues (`gosec ./...`)
- [ ] Code formatted (`make fmt`)
- [ ] Commit messages follow conventional format
- [ ] Documentation updated (if applicable)

### Running Full CI Locally

```bash
make ci-local
```

This runs the complete CI pipeline before pushing.

---

## Testing Requirements

### Test Coverage

- **New Features**: 80%+ coverage required
- **Bug Fixes**: Test must reproduce the bug
- **Refactoring**: Maintain or improve coverage

### Test Structure

Use table-driven tests for multiple scenarios:

```go
func TestWeatherService(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    float64
        wantErr bool
    }{
        {"valid input", "25.5", 25.5, false},
        {"invalid input", "invalid", 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseTemperature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseTemperature() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ParseTemperature() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Types

1. **Unit Tests**: Test individual functions/methods
   ```bash
   make test
   ```

2. **Integration Tests**: Test component interactions
   ```bash
   make test-integration
   ```

3. **End-to-End Tests**: Test full user flows
   ```bash
   make test-e2e
   ```

### Mocking

- Use `mockgen` for interface mocking
- Mock external dependencies (API calls, database)
- Don't mock simple types or structs

---

## Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions/improvements
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `style`: Code style changes (formatting)
- `chore`: Build/tooling changes
- `ci`: CI/CD changes

### Examples

```bash
# Feature
feat(alerts): add custom notification templates

# Bug fix
fix(weather): correct temperature conversion for Fahrenheit

# Documentation
docs(api): update weather service API documentation

# Breaking change
feat(auth)!: migrate to OAuth2 authentication

BREAKING CHANGE: API authentication now requires OAuth2 tokens
```

### Scope (Optional)

Scope indicates the area of the codebase:
- `alerts`: Alert system
- `weather`: Weather services
- `notifications`: Notification system
- `database`: Database operations
- `api`: API layer
- `ui`: User interface
- `docs`: Documentation

---

## Pull Request Process

### PR Template

When creating a PR, provide:

1. **Description**: What does this PR do?
2. **Motivation**: Why is this change needed?
3. **Changes**: List of specific changes
4. **Testing**: How was this tested?
5. **Screenshots**: For UI changes (if applicable)
6. **Checklist**: PR checklist completed

### PR Checklist

- [ ] Tests added/updated and passing
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (for user-facing changes)
- [ ] No linter warnings
- [ ] Coverage maintained or improved
- [ ] Commit messages follow conventional format
- [ ] Branch is up to date with `main`

### Review Process

1. **Automated Checks**: CI/CD runs automatically
2. **Code Review**: Maintainer reviews your changes
3. **Feedback**: Address review comments
4. **Approval**: PR approved by maintainer
5. **Merge**: PR merged to main branch

### Review Timeline

- **Simple fixes**: 1-2 days
- **Features**: 3-5 days
- **Major changes**: 1-2 weeks

---

## Documentation

### Types of Documentation

1. **Code Comments**: GoDoc for exported functions
   ```go
   // GetWeather retrieves current weather for the specified location.
   // Returns weather data and any error encountered.
   func GetWeather(ctx context.Context, location string) (*Weather, error)
   ```

2. **README Updates**: For user-facing changes
3. **API Documentation**: For service layer changes
4. **Architecture Docs**: For design changes

### Documentation Standards

- **Clear and Concise**: Write for clarity
- **Examples**: Include code examples
- **Complete**: Cover all parameters and return values
- **Updated**: Keep documentation synchronized with code

### Where to Document

- **User Guides**: `docs/` directory
- **API Reference**: `docs/API_REFERENCE.md`
- **Architecture**: `docs/ARCHITECTURE.md`
- **Configuration**: `docs/CONFIGURATION.md`

---

## Community

### Ways to Contribute

#### Code Contributions

- **Bug Fixes**: Fix reported issues
- **Features**: Implement new functionality
- **Tests**: Improve test coverage
- **Performance**: Optimize slow code

#### Non-Code Contributions

- **Documentation**: Improve guides, add examples
- **Translation**: Add/improve language support
- **Testing**: Report bugs, test beta releases
- **Design**: UI/UX improvements

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and ideas
- **Pull Requests**: Code contributions
- **Issues:** <https://github.com/valpere/shopogoda/issues>

### Feature Requests

To request a new feature:

1. **Check Existing Issues**: Avoid duplicates
2. **Open Discussion**: Describe use case
3. **Community Feedback**: Gather input
4. **Implementation**: Create PR if approved

### Reporting Bugs

When reporting bugs, include:

1. **Description**: What went wrong?
2. **Steps to Reproduce**: How to trigger the bug?
3. **Expected Behavior**: What should happen?
4. **Actual Behavior**: What actually happened?
5. **Environment**: OS, Go version, deployment platform
6. **Logs**: Relevant error messages

---

## Project Structure

Understanding the codebase:

```
shopogoda/
├── cmd/bot/              # Application entry points
├── internal/             # Private application code
│   ├── bot/             # Bot initialization
│   ├── config/          # Configuration management
│   ├── database/        # Database connections
│   ├── handlers/        # Telegram command handlers
│   ├── middleware/      # Bot middleware
│   ├── models/          # Data models
│   └── services/        # Business logic services
├── pkg/                 # Public libraries
│   ├── weather/         # Weather API clients
│   ├── alerts/          # Alert engine
│   └── metrics/         # Prometheus metrics
├── docs/                # Documentation
├── tests/               # Integration and E2E tests
└── deployments/         # Docker and K8s configs
```

---

## Design Principles

ShoPogoda adheres to:

### Core Principles

1. **DRY (Don't Repeat Yourself)**: Avoid code duplication
2. **YAGNI (You Aren't Gonna Need It)**: Implement only what's needed
3. **KISS (Keep It Simple, Stupid)**: Prefer simple solutions
4. **Encapsulation**: Hide implementation details
5. **PoLA (Principle of Least Astonishment)**: Design intuitive APIs

### SOLID Principles

- **Single Responsibility**: Each type has one clear purpose
- **Open/Closed**: Open for extension, closed for modification
- **Liskov Substitution**: Derived types must be substitutable
- **Interface Segregation**: Many specific interfaces over one general
- **Dependency Inversion**: Depend on abstractions, not concretions

---

## Release Process

ShoPogoda follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (x.0.0): Breaking changes
- **MINOR** (0.x.0): New features, backwards-compatible
- **PATCH** (0.0.x): Bug fixes, security patches

### Release Cycle

- **Major versions**: Every 6-9 months
- **Minor versions**: Every 2-3 months
- **Patch versions**: As needed

---

## Getting Help

- **Documentation**: Start with [README.md](README.md) and [docs/](docs/)
- **GitHub Issues**: Search existing issues
- **GitHub Discussions**: Ask questions
- **Issues:** <https://github.com/valpere/shopogoda/issues>

---

## Recognition

Contributors are recognized in:
- **CHANGELOG.md**: Listed in release notes
- **GitHub Contributors**: Automatic recognition
- **Project README**: Major contributors highlighted

---

## License

By contributing to ShoPogoda, you agree that your contributions will be licensed under the MIT License.

---

## Additional Resources

- **[Code Quality Guide](docs/CODE_QUALITY.md)**: Detailed quality standards
- **[Testing Guide](docs/TESTING.md)**: Comprehensive testing documentation
- **[API Reference](docs/API_REFERENCE.md)**: Service layer API documentation
- **[Roadmap](docs/ROADMAP.md)**: Project roadmap and future plans
- **[Deployment Guide](docs/DEPLOYMENT.md)**: Production deployment instructions

---

**Thank you for contributing to ShoPogoda!** 🌦️

Every contribution, no matter how small, helps make ShoPogoda better for everyone.
