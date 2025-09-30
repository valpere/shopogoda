package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Clear any previously registered metrics
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	m := New()

	assert.NotNil(t, m)
	assert.NotNil(t, m.counters)
	assert.NotNil(t, m.histograms)
	assert.NotNil(t, m.gauges)

	// Verify expected metrics are registered
	assert.Contains(t, m.counters, "bot_updates_total")
	assert.Contains(t, m.counters, "bot_errors_total")
	assert.Contains(t, m.counters, "weather_requests_total")

	assert.Contains(t, m.histograms, "bot_handler_duration_seconds")
	assert.Contains(t, m.histograms, "weather_api_duration_seconds")

	assert.Contains(t, m.gauges, "active_users")
	assert.Contains(t, m.gauges, "cache_hit_rate")
}

func TestMetrics_IncrementCounter(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	t.Run("increment existing counter", func(t *testing.T) {
		// Should not panic
		m.IncrementCounter("bot_updates_total", "message")
		assert.True(t, true)
	})

	t.Run("increment non-existent counter does not panic", func(t *testing.T) {
		// Should not panic for non-existent counter
		m.IncrementCounter("nonexistent_counter", "test")
		assert.True(t, true)
	})
}

func TestMetrics_ObserveHistogram(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	t.Run("observe existing histogram", func(t *testing.T) {
		// Should not panic
		m.ObserveHistogram("bot_handler_duration_seconds", 0.5, "weather")
		assert.True(t, true)
	})

	t.Run("observe non-existent histogram does not panic", func(t *testing.T) {
		// Should not panic for non-existent histogram
		m.ObserveHistogram("nonexistent_histogram", 1.0, "test")
		assert.True(t, true)
	})
}

func TestMetrics_SetGauge(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	t.Run("set existing gauge", func(t *testing.T) {
		// Should not panic
		m.SetGauge("active_users", 42.0)
		assert.True(t, true)
	})

	t.Run("set non-existent gauge does not panic", func(t *testing.T) {
		// Should not panic for non-existent gauge
		m.SetGauge("nonexistent_gauge", 1.0, "test")
		assert.True(t, true)
	})
}

func TestMetrics_Handler(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	handler := m.Handler()
	assert.NotNil(t, handler)
}

func TestMetrics_AllCounters(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	// Test all defined counters don't panic
	m.IncrementCounter("bot_updates_total", "message")
	m.IncrementCounter("bot_errors_total", "handler_error")
	m.IncrementCounter("weather_requests_total", "openweather", "success")

	assert.True(t, true)
}

func TestMetrics_AllHistograms(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	// Test all defined histograms don't panic
	m.ObserveHistogram("bot_handler_duration_seconds", 0.123, "command")
	m.ObserveHistogram("weather_api_duration_seconds", 0.456, "current")

	assert.True(t, true)
}

func TestMetrics_AllGauges(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	// Test all defined gauges don't panic
	m.SetGauge("active_users", 100.0)
	m.SetGauge("cache_hit_rate", 85.5, "redis")

	assert.True(t, true)
}
