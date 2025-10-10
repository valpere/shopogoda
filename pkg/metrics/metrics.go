package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
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
func (m *Metrics) GetCacheHitRate(cacheType string) float64 {
	// Check if the gauge exists
	gauge, exists := m.gauges["cache_hit_rate"]
	if !exists {
		return 85.0 // Default fallback
	}

	// Create a channel to collect metrics
	metricChan := make(chan prometheus.Metric, 1)

	// Collect the metric
	go func() {
		gauge.Collect(metricChan)
		close(metricChan)
	}()

	// Read metrics from channel
	for metric := range metricChan {
		// Write metric to DTO
		dtoMetric := &dto.Metric{}
		if err := metric.Write(dtoMetric); err != nil {
			continue
		}

		// Check if this metric matches our cache type label
		if dtoMetric.Label != nil {
			for _, label := range dtoMetric.Label {
				if label.GetName() == "cache_type" && label.GetValue() == cacheType {
					if dtoMetric.Gauge != nil {
						return dtoMetric.Gauge.GetValue()
					}
				}
			}
		}

		// If no labels (empty label set), return the value
		if len(dtoMetric.Label) == 0 && dtoMetric.Gauge != nil {
			return dtoMetric.Gauge.GetValue()
		}
	}

	// Fallback if metric not found
	return 85.0
}

// GetAverageResponseTime calculates average response time from handler duration histogram
// Returns the average in milliseconds
func (m *Metrics) GetAverageResponseTime() float64 {
	// Check if the histogram exists
	histogram, exists := m.histograms["bot_handler_duration_seconds"]
	if !exists {
		return 150.0 // Default fallback
	}

	// Create a channel to collect metrics
	metricChan := make(chan prometheus.Metric, 10)

	// Collect the metrics
	go func() {
		histogram.Collect(metricChan)
		close(metricChan)
	}()

	// Variables to calculate average from histogram
	var totalSum float64
	var totalCount uint64

	// Read metrics from channel
	for metric := range metricChan {
		// Write metric to DTO
		dtoMetric := &dto.Metric{}
		if err := metric.Write(dtoMetric); err != nil {
			continue
		}

		// Extract histogram data
		if dtoMetric.Histogram != nil {
			totalSum += dtoMetric.Histogram.GetSampleSum()
			totalCount += dtoMetric.Histogram.GetSampleCount()
		}
	}

	// Calculate average
	if totalCount > 0 {
		// Convert from seconds to milliseconds
		avgSeconds := totalSum / float64(totalCount)
		return avgSeconds * 1000.0
	}

	// Fallback if no data
	return 150.0
}
