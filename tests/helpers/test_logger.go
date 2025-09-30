package helpers

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog"
)

// TestLogger provides logging utilities for tests
type TestLogger struct {
	Buffer *bytes.Buffer
	Logger *zerolog.Logger
}

// NewTestLogger creates a new test logger that captures output
func NewTestLogger() *TestLogger {
	buffer := &bytes.Buffer{}
	logger := zerolog.New(buffer).With().Timestamp().Logger()

	return &TestLogger{
		Buffer: buffer,
		Logger: &logger,
	}
}

// NewTestLoggerWithLevel creates a test logger with specified level
func NewTestLoggerWithLevel(level zerolog.Level) *TestLogger {
	buffer := &bytes.Buffer{}
	logger := zerolog.New(buffer).Level(level).With().Timestamp().Logger()

	return &TestLogger{
		Buffer: buffer,
		Logger: &logger,
	}
}

// NewSilentTestLogger creates a logger that discards all output
func NewSilentTestLogger() *zerolog.Logger {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()
	return &logger
}

// NewConsoleTestLogger creates a logger that outputs to console for debugging
func NewConsoleTestLogger() *zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}
	logger := zerolog.New(output).Level(zerolog.DebugLevel).With().Timestamp().Logger()
	return &logger
}

// GetLogOutput returns the captured log output
func (tl *TestLogger) GetLogOutput() string {
	return tl.Buffer.String()
}

// Reset clears the log buffer
func (tl *TestLogger) Reset() {
	tl.Buffer.Reset()
}

// ContainsLog checks if the log buffer contains the specified string
func (tl *TestLogger) ContainsLog(t *testing.T, message string) bool {
	return bytes.Contains(tl.Buffer.Bytes(), []byte(message))
}

// AssertLogContains asserts that the log buffer contains the specified string
func (tl *TestLogger) AssertLogContains(t *testing.T, message string) {
	if !tl.ContainsLog(t, message) {
		t.Errorf("Expected log to contain '%s', but got: %s", message, tl.GetLogOutput())
	}
}

// AssertLogLevel asserts that a log entry with the specified level exists
func (tl *TestLogger) AssertLogLevel(t *testing.T, level string) {
	levelStr := `"level":"` + level + `"`
	if !bytes.Contains(tl.Buffer.Bytes(), []byte(levelStr)) {
		t.Errorf("Expected log to contain level '%s', but got: %s", level, tl.GetLogOutput())
	}
}
