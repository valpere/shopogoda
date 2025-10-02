package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {
	// Set test values
	Version = "1.0.0"
	GitCommit = "abc123def456"
	BuildTime = "2025-01-02_12:00:00_UTC"

	info := GetInfo()

	if info.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", info.Version)
	}

	if info.GitCommit != "abc123def456" {
		t.Errorf("Expected commit abc123def456, got %s", info.GitCommit)
	}

	if info.BuildTime != "2025-01-02_12:00:00_UTC" {
		t.Errorf("Expected build time 2025-01-02_12:00:00_UTC, got %s", info.BuildTime)
	}

	if info.GoVersion != runtime.Version() {
		t.Errorf("Expected Go version %s, got %s", runtime.Version(), info.GoVersion)
	}
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version:   "1.0.0",
		GitCommit: "abc123",
		BuildTime: "2025-01-02",
		GoVersion: "go1.24.6",
	}

	result := info.String()

	// Check all components are present
	expectedParts := []string{
		"ShoPogoda v1.0.0",
		"Commit: abc123",
		"Built: 2025-01-02",
		"Go: go1.24.6",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("String() output missing expected part: %s\nGot: %s", part, result)
		}
	}
}

func TestInfoShort(t *testing.T) {
	info := Info{
		Version:   "1.0.0",
		GitCommit: "abc123def456",
		BuildTime: "2025-01-02",
		GoVersion: "go1.24.6",
	}

	result := info.Short()
	expected := "v1.0.0 (abc123d)"

	if result != expected {
		t.Errorf("Short() = %s, want %s", result, expected)
	}
}

func TestInfoShortWithShortCommit(t *testing.T) {
	// Test edge case where commit is shorter than 7 characters
	info := Info{
		Version:   "0.1.0",
		GitCommit: "abc",
		BuildTime: "2025-01-02",
		GoVersion: "go1.24.6",
	}

	// This should not panic even with short commit
	result := info.Short()
	expected := "v0.1.0 (abc)"

	if result != expected {
		t.Errorf("Short() = %s, want %s", result, expected)
	}
}

func TestDefaultValues(t *testing.T) {
	// Reset to default values
	Version = "0.1.0-dev"
	GitCommit = "unknown"
	BuildTime = "unknown"

	info := GetInfo()

	if info.Version != "0.1.0-dev" {
		t.Errorf("Default version should be 0.1.0-dev, got %s", info.Version)
	}

	if info.GitCommit != "unknown" {
		t.Errorf("Default commit should be unknown, got %s", info.GitCommit)
	}

	if info.BuildTime != "unknown" {
		t.Errorf("Default build time should be unknown, got %s", info.BuildTime)
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
}
