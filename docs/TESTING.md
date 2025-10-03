# Testing Guide

Comprehensive testing guide for ShoPogoda, covering unit tests, integration tests, bot mocking, and coverage reporting.

## Table of Contents

- [Overview](#overview)
- [Test Coverage](#test-coverage)
- [Running Tests](#running-tests)
- [Test Types](#test-types)
- [Bot Mocking Infrastructure](#bot-mocking-infrastructure)
- [Writing Tests](#writing-tests)
- [Coverage Analysis](#coverage-analysis)
- [CI/CD Integration](#cicd-integration)

## Overview

ShoPogoda uses a comprehensive testing strategy with multiple test types:

- **Unit Tests**: Fast, isolated tests for individual functions and methods
- **Integration Tests**: Tests with real database and Redis using testcontainers
- **Bot Mock Tests**: Tests for Telegram bot handler functions using mock infrastructure

### Current Test Coverage

- **Overall**: 30.5%
- **Services Package**: 75.6% (core business logic)
- **Handlers Package**: 4.2% (bot command handlers)
- **Tests/Helpers Package**: 24.5% (test infrastructure)

## Test Coverage

### Coverage by Package

| Package | Coverage | Test Files |
|---------|----------|------------|
| `internal/services` | 75.6% | `*_test.go`, `tests/integration/*_test.go` |
| `internal/handlers/commands` | 4.2% | `commands_test.go` |
| `internal/models` | 2.6% | `models_test.go` |
| `internal/database` | 0.2% | `database_test.go` |
| `tests/helpers` | 24.5% | `bot_mock_test.go` |

### Coverage Target

- **Short-term**: 30% (✅ Achieved: 30.5%)
- **Medium-term**: 40%
- **Long-term**: 80%

## Running Tests

### Quick Test Commands

```bash
# Run all unit tests
make test

# Run tests with coverage report
make test-coverage

# Run integration tests (requires Docker)
make test-integration

# Run specific package tests
go test ./internal/services/... -v

# Run tests with race detection
go test -race ./...

# Run tests with verbose output
go test -v ./...
```

### Coverage Report Generation

```bash
# Generate HTML coverage report
make test-coverage

# Open coverage report in browser (Linux)
xdg-open coverage.html

# View coverage by function
go tool cover -func=coverage.tmp

# View coverage summary
go tool cover -func=coverage.tmp | grep total
```

### Test Output

```bash
# Run tests with coverage profiling
go test -coverprofile=coverage.tmp -coverpkg=./... ./...

# Generate HTML report
go tool cover -html=coverage.tmp -o coverage.html

# View coverage percentage
go tool cover -func=coverage.tmp | grep total
# Output: total: (statements) 30.5%
```

## Test Types

### 1. Unit Tests

**Location**: Alongside source files (`*_test.go`)

**Purpose**: Test individual functions and methods in isolation

**Example**: `internal/handlers/commands/commands_test.go`

```go
func TestGetAQIDescription(t *testing.T) {
    logger := helpers.NewSilentTestLogger()
    locService := services.NewLocalizationService(logger)
    svc := &services.Services{
        Localization: locService,
    }

    handler := &CommandHandler{
        services: svc,
        logger:   logger,
    }

    tests := []struct {
        name     string
        aqi      int
        expected string
    }{
        {"Good air quality - 0", 0, "aqi_good"},
        {"Good air quality - 50", 50, "aqi_good"},
        {"Moderate - 51", 51, "aqi_moderate"},
        {"Unhealthy - 151", 151, "aqi_unhealthy"},
        {"Hazardous - 301", 301, "aqi_hazardous"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := handler.getAQIDescription(tt.aqi, "en")
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Characteristics**:
- Fast execution (milliseconds)
- No external dependencies
- Table-driven test pattern
- Uses testify/assert for assertions

### 2. Integration Tests

**Location**: `tests/integration/*_test.go`

**Purpose**: Test service interactions with real database and Redis

**Example**: `tests/integration/user_service_test.go`

```go
func TestIntegration_UserServiceRegisterUser(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    testDB, testRedis, cleanup := helpers.SetupTestEnvironment(t)
    defer cleanup()

    logger := helpers.NewSilentTestLogger()
    userService := services.NewUserService(testDB.DB, testRedis.Client, logger)

    t.Run("register new user", func(t *testing.T) {
        ctx := context.Background()
        userID := int64(12345)

        err := userService.RegisterUser(ctx, userID, "testuser", "Test", "User")
        assert.NoError(t, err)

        // Verify user in database
        var user models.User
        err = testDB.DB.Where("telegram_id = ?", userID).First(&user).Error
        assert.NoError(t, err)
        assert.Equal(t, "testuser", user.Username)
    })
}
```

**Characteristics**:
- Uses testcontainers for PostgreSQL and Redis
- Tests real database transactions
- Tests Redis caching behavior
- Slower execution (seconds)
- Skipped in short mode (`go test -short`)

### 3. Bot Mock Tests

**Location**: `tests/helpers/bot_mock_test.go`, `internal/handlers/commands/commands_test.go`

**Purpose**: Test Telegram bot handler functions without real bot

**Example**: `internal/handlers/commands/commands_test.go`

```go
func TestParseLocationFromArgs(t *testing.T) {
    handler := &CommandHandler{}

    tests := []struct {
        name     string
        args     []string
        expected string
    }{
        {"No args", []string{"/weather"}, ""},
        {"Single word location", []string{"/weather", "London"}, "London"},
        {"Multi-word location", []string{"/weather", "New", "York"}, "New York"},
        {"Location with comma", []string{"/weather", "Paris,", "France"}, "Paris, France"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
                Args: tt.args,
            })

            result := handler.parseLocationFromArgs(mockCtx.Context)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Characteristics**:
- Uses bot mocking infrastructure
- Tests handler functions in isolation
- Fast execution like unit tests
- No external dependencies

## Bot Mocking Infrastructure

### Overview

ShoPogoda provides comprehensive bot mocking infrastructure for testing Telegram bot handler functions without requiring a real bot instance.

**Location**: `tests/helpers/bot_mock.go`

### MockBot

Creates minimal `gotgbot.Bot` instances for testing.

```go
// Create a mock bot
mockBot := helpers.NewMockBot()

// Use in tests
assert.NotNil(t, mockBot.Bot)
assert.Equal(t, int64(12345), mockBot.Bot.Id)
assert.Equal(t, "TestBot", mockBot.Bot.FirstName)
```

### MockContext

Creates flexible `ext.Context` instances with configurable options.

**Available Fields**:
- `UserID` - Telegram user ID (default: 12345)
- `Username` - Username (default: "testuser")
- `FirstName` - First name (default: "Test")
- `LastName` - Last name
- `ChatID` - Chat ID (default: 12345)
- `MessageID` - Message ID (default: 1)
- `MessageText` - Message text
- `Args` - Command arguments (parsed from message text)
- `CallbackID` - Callback query ID
- `Data` - Callback data
- `Latitude` - Location latitude
- `Longitude` - Location longitude

### Usage Examples

#### Basic Context

```go
// Create simple mock context
mockCtx := helpers.NewSimpleMockContext(12345, "/weather London")

// Access context
ctx := mockCtx.Context
assert.Equal(t, int64(12345), ctx.EffectiveUser.Id)
assert.Equal(t, "/weather London", ctx.EffectiveMessage.Text)
```

#### Context with Arguments

```go
// Create context with command arguments
mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
    Args: []string{"/weather", "New", "York"},
})

// Args are automatically parsed from joined text
args := mockCtx.Context.Args()
assert.Equal(t, []string{"/weather", "New", "York"}, args)
```

#### Context with Location

```go
// Create context with GPS location
mockCtx := helpers.NewMockContextWithLocation(12345, 40.7128, -74.0060)

// Access location
loc := mockCtx.Context.EffectiveMessage.Location
assert.Equal(t, 40.7128, loc.Latitude)
assert.Equal(t, -74.0060, loc.Longitude)
```

#### Context with Callback Query

```go
// Create context with callback data
mockCtx := helpers.NewMockContextWithCallback(12345, "callback_123", "action:confirm")

// Access callback query
cb := mockCtx.Context.CallbackQuery
assert.Equal(t, "callback_123", cb.Id)
assert.Equal(t, "action:confirm", cb.Data)
```

#### Custom Context Options

```go
// Create fully customized context
mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
    UserID:      99999,
    Username:    "customuser",
    FirstName:   "Custom",
    LastName:    "User",
    ChatID:      88888,
    MessageID:   777,
    MessageText: "Custom message",
})
```

### Context.Args() Compatibility

The mock infrastructure properly handles `Context.Args()` by synchronizing `Update.Message.Text` and `EffectiveMessage.Text`:

```go
// When Args are provided
mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
    Args: []string{"/weather", "New", "York"},
})

// Both fields are set to "New York"
assert.Equal(t, "/weather New York", mockCtx.Context.Update.Message.Text)
assert.Equal(t, "/weather New York", mockCtx.Context.EffectiveMessage.Text)

// Args() parses correctly
args := mockCtx.Context.Args()
assert.Equal(t, []string{"/weather", "New", "York"}, args)
```

## Writing Tests

### Test File Organization

```
internal/
├── handlers/
│   ├── commands/
│   │   ├── commands.go
│   │   └── commands_test.go          # Unit tests for handler functions
│   └── ...
├── services/
│   ├── user_service.go
│   ├── user_service_test.go          # Unit tests with mocks
│   └── ...
tests/
├── integration/
│   ├── user_service_test.go          # Integration tests with real DB/Redis
│   ├── weather_service_test.go
│   └── ...
├── helpers/
│   ├── bot_mock.go                   # Bot mocking infrastructure
│   ├── bot_mock_test.go              # Tests for mock infrastructure
│   ├── test_logger.go                # Silent logger for tests
│   └── test_environment.go           # Test environment setup
└── fixtures/
    └── ...                           # Test data fixtures
```

### Test Naming Conventions

```go
// Unit tests
func TestFunctionName(t *testing.T) {}
func TestStructMethod(t *testing.T) {}

// Integration tests
func TestIntegration_ServiceName_MethodName(t *testing.T) {}

// Table-driven subtests
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"description", "input", "output"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

### Test Best Practices

1. **Use Table-Driven Tests** for multiple scenarios
```go
tests := []struct {
    name     string
    input    int
    expected string
}{
    {"zero", 0, "zero"},
    {"positive", 5, "five"},
    {"negative", -1, "negative"},
}
```

2. **Use testify/assert** for readable assertions
```go
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.NotNil(t, result)
```

3. **Use subtests** for organization
```go
t.Run("success case", func(t *testing.T) {
    // Test logic
})
```

4. **Clean up resources** with defer
```go
testDB, testRedis, cleanup := helpers.SetupTestEnvironment(t)
defer cleanup()
```

5. **Skip integration tests in short mode**
```go
if testing.Short() {
    t.Skip("Skipping integration test")
}
```

## Coverage Analysis

### Package Coverage Breakdown

```bash
# View coverage by package
go tool cover -func=coverage.tmp | grep -v "100.0%$" | sort -k3 -n

# View top uncovered functions
go tool cover -func=coverage.tmp | grep "0.0%$"

# View coverage for specific package
go tool cover -func=coverage.tmp | grep "internal/services"
```

### Coverage Reports

**HTML Report**: `coverage.html` (generated by `make test-coverage`)

Open in browser to see:
- Line-by-line coverage highlighting
- Coverage percentage per file
- Navigation between files

**Terminal Report**:
```bash
go tool cover -func=coverage.tmp

# Output:
# github.com/valpere/shopogoda/internal/services/user_service.go:45:    RegisterUser      100.0%
# github.com/valpere/shopogoda/internal/services/user_service.go:67:    GetUser           100.0%
# ...
# total:                                                                (statements)      30.5%
```

### Coverage by Test Type

- **Unit Tests**: ~15% overall coverage (fast, focused)
- **Integration Tests**: ~10% overall coverage (comprehensive service testing)
- **Bot Mock Tests**: ~5% overall coverage (handler functions)

## CI/CD Integration

### GitHub Actions

Tests run automatically on:
- Pull requests
- Pushes to main/develop branches
- Manual workflow dispatch

### CI Test Configuration

**.github/workflows/test.yml**:
```yaml
- name: Run tests with coverage
  run: |
    go test -v -coverprofile=coverage.tmp -coverpkg=./... ./...
    go tool cover -func=coverage.tmp

- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v5
  with:
    file: ./coverage.tmp
    flags: unittests
    name: codecov-umbrella
```

### Coverage Reporting

- **Codecov**: Automatic coverage reports on PRs
- **Coverage Badge**: Shows current coverage percentage
- **Coverage Trends**: Track coverage over time

### Test Validation

All PRs must pass:
- ✅ Unit tests
- ✅ Integration tests (if applicable)
- ✅ Linting (golangci-lint)
- ✅ Formatting (gofmt)
- ✅ Security scans

## Test Helpers

### Silent Test Logger

```go
logger := helpers.NewSilentTestLogger()
```

Creates a zerolog logger that discards all output (for clean test output).

### Test Environment Setup

```go
testDB, testRedis, cleanup := helpers.SetupTestEnvironment(t)
defer cleanup()

// Use testDB.DB and testRedis.Client in tests
```

Sets up PostgreSQL and Redis containers using testcontainers.

### Mock Factories

```go
// Create mock bot
mockBot := helpers.NewMockBot()

// Create mock context (various options)
mockCtx := helpers.NewMockContext(helpers.MockContextOptions{...})
mockCtx := helpers.NewSimpleMockContext(userID, text)
mockCtx := helpers.NewMockContextWithLocation(userID, lat, lon)
mockCtx := helpers.NewMockContextWithCallback(userID, cbID, data)
```

## Troubleshooting

### Common Issues

**Issue**: Tests fail with "database connection refused"
```bash
# Solution: Start test containers
docker-compose up -d postgres redis
```

**Issue**: Coverage report shows 0%
```bash
# Solution: Use -coverpkg flag
go test -coverprofile=coverage.tmp -coverpkg=./... ./...
```

**Issue**: Integration tests timeout
```bash
# Solution: Increase timeout
go test -timeout 5m ./tests/integration/...
```

**Issue**: Mock context Args() returns nil
```bash
# Solution: Use Args field in MockContextOptions
mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
    Args: []string{"/command", "arg1", "arg2"},
})
```

## Future Improvements

- [ ] Increase handler coverage to 20% (current: 4.2%)
- [ ] Add E2E tests with real Telegram API
- [ ] Add benchmark tests for performance tracking
- [ ] Add mutation testing for test quality validation
- [ ] Add fuzzing tests for input validation
- [ ] Create test data factories for complex models
- [ ] Add API integration tests with mock HTTP server

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Testcontainers Go](https://golang.testcontainers.org/)
- [Go Coverage Tool](https://go.dev/blog/cover)

---

**Last Updated**: 2025-01-03
**Maintained by**: [@valpere](https://github.com/valpere)
