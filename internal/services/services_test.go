package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/pkg/metrics"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestNew(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	mockRedis := helpers.NewMockRedis()
	logger := helpers.NewSilentTestLogger()

	cfg := &config.Config{
		Weather: config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
		},
		Integrations: config.IntegrationsConfig{
			SlackWebhookURL: "https://hooks.slack.com/test",
		},
	}

	metricsCollector := metrics.New()
	services := New(mockDB.DB, mockRedis.Client, cfg, logger, metricsCollector)

	assert.NotNil(t, services)
	assert.NotNil(t, services.User)
	assert.NotNil(t, services.Weather)
	assert.NotNil(t, services.Alert)
	assert.NotNil(t, services.Subscription)
	assert.NotNil(t, services.Notification)
	assert.NotNil(t, services.Scheduler)
	assert.NotNil(t, services.Export)
	assert.NotNil(t, services.Localization)
}

func TestServices_StartScheduler(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	mockRedis := helpers.NewMockRedis()
	logger := helpers.NewSilentTestLogger()

	cfg := &config.Config{
		Weather: config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
		},
		Integrations: config.IntegrationsConfig{
			SlackWebhookURL: "https://hooks.slack.com/test",
		},
	}

	metricsCollector := metrics.New()
	services := New(mockDB.DB, mockRedis.Client, cfg, logger, metricsCollector)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start scheduler in goroutine
	go services.StartScheduler(ctx)

	// Cancel immediately to stop the scheduler
	cancel()

	// Verify scheduler can be stopped
	services.Stop()

	// Test should complete without hanging
}

func TestServices_Stop(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	mockRedis := helpers.NewMockRedis()
	logger := helpers.NewSilentTestLogger()

	cfg := &config.Config{
		Weather: config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
		},
		Integrations: config.IntegrationsConfig{
			SlackWebhookURL: "https://hooks.slack.com/test",
		},
	}

	metricsCollector := metrics.New()
	services := New(mockDB.DB, mockRedis.Client, cfg, logger, metricsCollector)

	// Stop should not panic even without starting scheduler
	assert.NotPanics(t, func() {
		services.Stop()
	})
}
