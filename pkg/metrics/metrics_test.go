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

func TestMetrics_GetCacheHitRate(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	t.Run("returns default when no data", func(t *testing.T) {
		rate := m.GetCacheHitRate("weather")
		assert.Equal(t, 85.0, rate)
	})

	t.Run("returns actual value after setting gauge", func(t *testing.T) {
		// Set a cache hit rate
		m.SetGauge("cache_hit_rate", 92.5, "weather")

		// Retrieve it
		rate := m.GetCacheHitRate("weather")
		assert.Equal(t, 92.5, rate)
	})

	t.Run("returns correct value for different cache types", func(t *testing.T) {
		m.SetGauge("cache_hit_rate", 88.0, "redis")
		m.SetGauge("cache_hit_rate", 95.5, "memory")

		redisRate := m.GetCacheHitRate("redis")
		memoryRate := m.GetCacheHitRate("memory")

		assert.Equal(t, 88.0, redisRate)
		assert.Equal(t, 95.5, memoryRate)
	})
}

func TestMetrics_GetAverageResponseTime(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m := New()

	t.Run("returns default when no observations", func(t *testing.T) {
		avgTime := m.GetAverageResponseTime()
		assert.Equal(t, 150.0, avgTime)
	})

	t.Run("calculates average from histogram observations", func(t *testing.T) {
		// Record some observations (in seconds)
		m.ObserveHistogram("bot_handler_duration_seconds", 0.100, "command") // 100ms
		m.ObserveHistogram("bot_handler_duration_seconds", 0.200, "command") // 200ms
		m.ObserveHistogram("bot_handler_duration_seconds", 0.300, "command") // 300ms

		// Average should be 200ms
		avgTime := m.GetAverageResponseTime()
		assert.InDelta(t, 200.0, avgTime, 1.0) // Allow 1ms delta for floating point
	})

	t.Run("calculates average across multiple handler types", func(t *testing.T) {
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
		m := New()

		// Record observations for different handler types
		m.ObserveHistogram("bot_handler_duration_seconds", 0.050, "weather")  // 50ms
		m.ObserveHistogram("bot_handler_duration_seconds", 0.150, "forecast") // 150ms
		m.ObserveHistogram("bot_handler_duration_seconds", 0.100, "air")      // 100ms

		// Average should be 100ms
		avgTime := m.GetAverageResponseTime()
		assert.InDelta(t, 100.0, avgTime, 1.0)
	})

	t.Run("updates average as more observations are added", func(t *testing.T) {
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
		m := New()

		// First observation
		m.ObserveHistogram("bot_handler_duration_seconds", 0.100, "test")
		avg1 := m.GetAverageResponseTime()
		assert.InDelta(t, 100.0, avg1, 1.0)

		// Add more observations
		m.ObserveHistogram("bot_handler_duration_seconds", 0.300, "test")
		avg2 := m.GetAverageResponseTime()
		assert.InDelta(t, 200.0, avg2, 1.0) // (100 + 300) / 2 = 200
	})
}
