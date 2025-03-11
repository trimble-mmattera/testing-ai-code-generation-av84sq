// Package metrics provides instrumentation capabilities for the Document Management Platform.
// It implements Prometheus-based metrics collection, registration, and exposure,
// enabling monitoring of system health, performance, and business operations.
// The package supports various metric types including counters, gauges, histograms, and summaries.
package metrics

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus" // v1.14.0+
	"github.com/prometheus/client_golang/prometheus/promauto" // v1.14.0+
	"github.com/prometheus/client_golang/prometheus/promhttp" // v1.14.0+

	"src/backend/pkg/logger"
)

var (
	registry    *prometheus.Registry
	initialized bool
	httpServer  *http.Server
	initLock    sync.Mutex
	namespace   = "document_mgmt"

	// HTTP metrics
	httpRequestsTotal    prometheus.Counter
	httpRequestDuration  prometheus.Histogram
	httpRequestsInFlight prometheus.Gauge

	// Document metrics
	documentUploadsTotal       prometheus.CounterVec
	documentDownloadsTotal     prometheus.CounterVec
	documentSearchesTotal      prometheus.Counter
	documentProcessingDuration prometheus.Histogram

	// Security metrics
	virusDetectionsTotal prometheus.Counter

	// Storage metrics
	storageUsageBytes prometheus.GaugeVec
)

// MetricsConfig defines configuration options for the metrics system
type MetricsConfig struct {
	// Enabled determines if metrics collection is enabled
	Enabled bool
	// EnableEndpoint determines if metrics HTTP endpoint is exposed
	EnableEndpoint bool
	// EndpointAddress is the address to bind the metrics HTTP server
	EndpointAddress string
	// EndpointPort is the port for the metrics HTTP server
	EndpointPort int
	// Namespace is the prefix for all metrics
	Namespace string
}

// NewMetricsConfig creates a new MetricsConfig with default values
func NewMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled:         true,
		EnableEndpoint:  true,
		EndpointAddress: "0.0.0.0",
		EndpointPort:    9090,
		Namespace:       "document_mgmt",
	}
}

// Timer is used for measuring operation duration
type Timer struct {
	operation string
	startTime time.Time
}

// NewTimer creates a new timer for measuring operation duration
func NewTimer(operation string) *Timer {
	return &Timer{
		operation: operation,
		startTime: time.Now(),
	}
}

// ObserveDuration records the duration since the timer was created
func (t *Timer) ObserveDuration() time.Duration {
	duration := time.Since(t.startTime)

	// Record duration based on operation type
	switch t.operation {
	case "http_request":
		ObserveHTTPRequestDuration(duration)
	case "document_processing":
		ObserveDocumentProcessingDuration(duration)
	}

	return duration
}

// Init initializes the metrics system with the specified configuration
func Init(config MetricsConfig) error {
	initLock.Lock()
	defer initLock.Unlock()

	// Check if already initialized
	if initialized {
		return nil
	}

	// Create a new registry
	registry = prometheus.NewRegistry()

	// Register default Go collectors
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())

	// Set custom namespace if provided
	if config.Namespace != "" {
		namespace = config.Namespace
	}

	// Initialize all metrics
	initializeMetrics()

	// Start HTTP server if endpoint is enabled
	if config.EnableEndpoint {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

		httpServer = &http.Server{
			Addr:    config.EndpointAddress + ":" + strconv.Itoa(config.EndpointPort),
			Handler: mux,
		}

		go func() {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Metrics HTTP server failed", "error", err)
			}
		}()

		logger.Info("Metrics HTTP server started",
			"address", config.EndpointAddress,
			"port", config.EndpointPort)
	}

	initialized = true
	logger.Info("Metrics system initialized", "namespace", namespace)
	return nil
}

// initializeMetrics creates and registers all metrics
func initializeMetrics() {
	// HTTP metrics
	httpRequestsTotal = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests",
	})

	httpRequestDuration = promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	})

	httpRequestsInFlight = promauto.With(registry).NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "http_requests_in_flight",
		Help:      "Current number of HTTP requests in flight",
	})

	// Document metrics
	documentUploadsTotal = *promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "document_uploads_total",
		Help:      "Total number of document uploads",
	}, []string{"tenant_id", "content_type"})

	documentDownloadsTotal = *promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "document_downloads_total",
		Help:      "Total number of document downloads",
	}, []string{"tenant_id", "content_type"})

	documentSearchesTotal = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "document_searches_total",
		Help:      "Total number of document searches",
	})

	documentProcessingDuration = promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "document_processing_duration_seconds",
		Help:      "Document processing duration in seconds",
		Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
	})

	// Security metrics
	virusDetectionsTotal = promauto.With(registry).NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "virus_detections_total",
		Help:      "Total number of virus detections",
	})

	// Storage metrics
	storageUsageBytes = *promauto.With(registry).NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "storage_usage_bytes",
		Help:      "Current storage usage in bytes",
	}, []string{"tenant_id", "bucket_type"})
}

// Shutdown stops the metrics system, closing the HTTP server if running
func Shutdown() error {
	initLock.Lock()
	defer initLock.Unlock()

	if !initialized {
		return nil
	}

	// Stop HTTP server if running
	if httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			return err
		}
		httpServer = nil
	}

	initialized = false
	logger.Info("Metrics system shut down")
	return nil
}

// Handler returns an HTTP handler for exposing metrics
func Handler() http.Handler {
	if !initialized {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Metrics system not initialized"))
		})
	}
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// IncHTTPRequests increments the HTTP requests counter
func IncHTTPRequests() {
	if !initialized {
		return
	}
	httpRequestsTotal.Inc()
}

// ObserveHTTPRequestDuration records the duration of an HTTP request
func ObserveHTTPRequestDuration(duration time.Duration) {
	if !initialized {
		return
	}
	httpRequestDuration.Observe(duration.Seconds())
}

// IncHTTPRequestsInFlight increments the gauge of in-flight HTTP requests
func IncHTTPRequestsInFlight() {
	if !initialized {
		return
	}
	httpRequestsInFlight.Inc()
}

// DecHTTPRequestsInFlight decrements the gauge of in-flight HTTP requests
func DecHTTPRequestsInFlight() {
	if !initialized {
		return
	}
	httpRequestsInFlight.Dec()
}

// IncDocumentUploads increments the document uploads counter
func IncDocumentUploads(tenantID, contentType string) {
	if !initialized {
		return
	}
	documentUploadsTotal.WithLabelValues(tenantID, contentType).Inc()
}

// IncDocumentDownloads increments the document downloads counter
func IncDocumentDownloads(tenantID, contentType string) {
	if !initialized {
		return
	}
	documentDownloadsTotal.WithLabelValues(tenantID, contentType).Inc()
}

// IncDocumentSearches increments the document searches counter
func IncDocumentSearches() {
	if !initialized {
		return
	}
	documentSearchesTotal.Inc()
}

// ObserveDocumentProcessingDuration records the duration of document processing
func ObserveDocumentProcessingDuration(duration time.Duration) {
	if !initialized {
		return
	}
	documentProcessingDuration.Observe(duration.Seconds())
}

// IncVirusDetections increments the virus detections counter
func IncVirusDetections() {
	if !initialized {
		return
	}
	virusDetectionsTotal.Inc()
}

// SetStorageUsage sets the current storage usage in bytes
func SetStorageUsage(tenantID, bucketType string, bytes float64) {
	if !initialized {
		return
	}
	storageUsageBytes.WithLabelValues(tenantID, bucketType).Set(bytes)
}

// RegisterCustomCounter registers a custom counter metric
func RegisterCustomCounter(name, help string, labelNames []string) *prometheus.CounterVec {
	if !initialized {
		return nil
	}
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      name,
		Help:      help,
	}, labelNames)
	registry.MustRegister(counter)
	return counter
}

// RegisterCustomGauge registers a custom gauge metric
func RegisterCustomGauge(name, help string, labelNames []string) *prometheus.GaugeVec {
	if !initialized {
		return nil
	}
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      name,
		Help:      help,
	}, labelNames)
	registry.MustRegister(gauge)
	return gauge
}

// RegisterCustomHistogram registers a custom histogram metric
func RegisterCustomHistogram(name, help string, labelNames []string, buckets []float64) *prometheus.HistogramVec {
	if !initialized {
		return nil
	}
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}, labelNames)
	registry.MustRegister(histogram)
	return histogram
}

// RegisterCustomSummary registers a custom summary metric
func RegisterCustomSummary(name, help string, labelNames []string, opts prometheus.SummaryOpts) *prometheus.SummaryVec {
	if !initialized {
		return nil
	}
	opts.Namespace = namespace
	opts.Name = name
	opts.Help = help
	summary := prometheus.NewSummaryVec(opts, labelNames)
	registry.MustRegister(summary)
	return summary
}