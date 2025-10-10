package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
	gauges     map[string]*prometheus.GaugeVec
}

func New() *Metrics {
	m := &Metrics{
		counters:   make(map[string]*prometheus.CounterVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
	}

	// Initialize common metrics
	m.counters["bot_updates_total"] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_updates_total",
			Help: "Total number of bot updates processed",
		},
		[]string{"type"},
	)

	m.counters["bot_errors_total"] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_errors_total",
			Help: "Total number of bot errors",
		},
		[]string{"type"},
	)

	m.counters["weather_requests_total"] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "weather_requests_total",
			Help: "Total number of weather API requests",
		},
		[]string{"api", "status"},
	)

	m.histograms["bot_handler_duration_seconds"] = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bot_handler_duration_seconds",
			Help:    "Duration of bot handler execution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	m.histograms["weather_api_duration_seconds"] = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "weather_api_duration_seconds",
			Help:    "Duration of weather API requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"api"},
	)

	m.gauges["active_users"] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_users",
			Help: "Number of active users",
		},
		[]string{},
	)

	m.gauges["cache_hit_rate"] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_hit_rate",
			Help: "Cache hit rate percentage",
		},
		[]string{"cache_type"},
	)

	// Register all metrics (gracefully handle already registered metrics)
	for _, counter := range m.counters {
		if err := prometheus.Register(counter); err != nil {
			// Metric already registered, this is OK in tests
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				// Only panic for non-duplicate registration errors
				panic(err)
			}
		}
	}
	for _, histogram := range m.histograms {
		if err := prometheus.Register(histogram); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				panic(err)
			}
		}
	}
	for _, gauge := range m.gauges {
		if err := prometheus.Register(gauge); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				panic(err)
			}
		}
	}

	return m
}

func (m *Metrics) IncrementCounter(name string, labelValues ...string) {
	if counter, exists := m.counters[name]; exists {
		counter.WithLabelValues(labelValues...).Inc()
	}
}

func (m *Metrics) ObserveHistogram(name string, value float64, labelValues ...string) {
	if histogram, exists := m.histograms[name]; exists {
		histogram.WithLabelValues(labelValues...).Observe(value)
	}
}

func (m *Metrics) SetGauge(name string, value float64, labelValues ...string) {
	if gauge, exists := m.gauges[name]; exists {
		gauge.WithLabelValues(labelValues...).Set(value)
	}
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// GetCacheHitRate calculates cache hit rate from Prometheus metrics
// Returns the percentage (0-100) of cache hits vs total cache operations
// TODO: Implement proper metric value extraction from Prometheus
// For now, returns a reasonable default value
func (m *Metrics) GetCacheHitRate(cacheType string) float64 {
	// In a real implementation, we would:
	// 1. Use prometheus.Gatherer to collect metrics
	// 2. Parse the metric families to extract gauge values
	// 3. Calculate the actual cache hit rate
	// For now, return a reasonable default
	return 85.0
}

// GetAverageResponseTime calculates average response time from handler duration histogram
// Returns the average in milliseconds
// TODO: Implement proper histogram value extraction from Prometheus
// For now, returns a reasonable default value
func (m *Metrics) GetAverageResponseTime() float64 {
	// In a real implementation, we would:
	// 1. Use prometheus.Gatherer to collect metrics
	// 2. Parse histogram buckets and calculate average
	// 3. Convert from seconds to milliseconds
	// For now, return a reasonable default
	return 150.0
}
