// Package tracing provides distributed tracing capabilities for the Document Management Platform.
// It implements OpenTelemetry-based tracing to track request flows across microservices,
// enabling end-to-end visibility of operations and performance monitoring.
// The package supports context propagation, span creation, and integration with various backends.
package tracing

import (
	"context"
	"fmt"
	"os"

	"../logger" // For logging tracing initialization and errors

	"go.opentelemetry.io/otel" // v1.11.0+
	"go.opentelemetry.io/otel/attribute" // v1.11.0+
	"go.opentelemetry.io/otel/codes" // v1.11.0+
	"go.opentelemetry.io/otel/exporters/jaeger" // v1.11.0+
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace" // v1.11.0+
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc" // v1.11.0+
	"go.opentelemetry.io/otel/propagation" // v1.11.0+
	"go.opentelemetry.io/otel/sdk/resource" // v1.11.0+
	sdktrace "go.opentelemetry.io/otel/sdk/trace" // v1.11.0+
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0" // v1.11.0+
	"go.opentelemetry.io/otel/trace" // v1.11.0+
)

// Global variables
var (
	tracer         trace.Tracer
	tracerProvider trace.TracerProvider
	initialized    bool
	// Default sampling ratio is 10% of requests
	defaultSamplingRatio = 0.1
)

// TracingConfig defines the configuration for the tracing system
type TracingConfig struct {
	// Enabled indicates whether tracing is enabled
	Enabled bool
	// ServiceName is the name of the service being traced
	ServiceName string
	// ServiceVersion is the version of the service
	ServiceVersion string
	// Environment is the deployment environment (e.g., development, staging, production)
	Environment string
	// ExporterType specifies the tracing backend (otlp, jaeger)
	ExporterType string
	// Endpoint is the URL of the tracing collector
	Endpoint string
	// SamplingRatio is the ratio of requests to sample (0.0 to 1.0)
	SamplingRatio float64
}

// NewTracingConfig creates a new TracingConfig with default values
func NewTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:        true,
		ServiceName:    "document-mgmt-platform",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   "jaeger",
		Endpoint:       "http://localhost:14268/api/traces",
		SamplingRatio:  defaultSamplingRatio,
	}
}

// Init initializes the tracing system with the specified configuration
func Init(config TracingConfig) error {
	// Check if already initialized
	if initialized {
		return nil
	}

	// If tracing is disabled, return early
	if !config.Enabled {
		logger.Info("Tracing is disabled")
		return nil
	}

	// Create a resource with service information
	res := createResource(config.ServiceName, config.ServiceVersion, config.Environment)

	// Create the appropriate exporter based on configuration
	exporter, err := createExporter(config)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Determine sampling ratio
	samplingRatio := config.SamplingRatio
	if samplingRatio <= 0 {
		samplingRatio = defaultSamplingRatio
	}

	// Create a trace provider with the exporter and sampling configuration
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(samplingRatio)),
	)

	// Set the global trace provider
	otel.SetTracerProvider(tp)

	// Set the global propagator to propagate trace context across services
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create a tracer with the service name
	tracer = tp.Tracer(config.ServiceName)
	tracerProvider = tp

	// Set initialized to true
	initialized = true

	logger.Info("Tracing initialized successfully",
		"service", config.ServiceName,
		"exporter", config.ExporterType,
		"sampling_ratio", samplingRatio)

	return nil
}

// Shutdown shuts down the tracing system, flushing any pending spans
func Shutdown() error {
	// Check if initialized
	if !initialized {
		return nil
	}

	// Type assert to get the SDK tracer provider for proper shutdown
	if tp, ok := tracerProvider.(*sdktrace.TracerProvider); ok {
		if err := tp.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
	}

	// Set initialized to false
	initialized = false

	logger.Info("Tracing shutdown successfully")
	return nil
}

// StartSpan starts a new span with the given name and parent context
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	// If not initialized, return a no-op span
	if !initialized {
		return ctx, trace.SpanFromContext(ctx)
	}

	// Start a new span with the given name and parent context
	return tracer.Start(ctx, name)
}

// EndSpan ends a span with optional status and attributes
func EndSpan(span trace.Span, err error) {
	// Check if span is nil
	if span == nil {
		return
	}

	// If error is not nil, set span status to error with message
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	// End the span
	span.End()
}

// AddAttribute adds an attribute to a span
func AddAttribute(span trace.Span, key string, value interface{}) {
	// Check if span is nil
	if span == nil {
		return
	}

	// Convert value to appropriate attribute type and set attribute on span
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	default:
		span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// AddEvent adds an event to a span
func AddEvent(span trace.Span, name string, attributes ...attribute.KeyValue) {
	// Check if span is nil
	if span == nil {
		return
	}

	// Add event with name and attributes to span
	span.AddEvent(name, trace.WithAttributes(attributes...))
}

// SpanFromContext extracts a span from the context
func SpanFromContext(ctx context.Context) trace.Span {
	// Check if context is nil
	if ctx == nil {
		return trace.SpanFromContext(context.Background())
	}

	// Extract span from context
	return trace.SpanFromContext(ctx)
}

// TraceIDFromContext extracts the trace ID from the context
func TraceIDFromContext(ctx context.Context) string {
	// Check if context is nil
	if ctx == nil {
		return ""
	}

	// Extract span from context
	span := trace.SpanFromContext(ctx)
	
	// Get span context
	spanContext := span.SpanContext()
	
	// If span context is not valid, return empty string
	if !spanContext.IsValid() {
		return ""
	}
	
	// Return trace ID as string
	return spanContext.TraceID().String()
}

// SpanIDFromContext extracts the span ID from the context
func SpanIDFromContext(ctx context.Context) string {
	// Check if context is nil
	if ctx == nil {
		return ""
	}

	// Extract span from context
	span := trace.SpanFromContext(ctx)
	
	// Get span context
	spanContext := span.SpanContext()
	
	// If span context is not valid, return empty string
	if !spanContext.IsValid() {
		return ""
	}
	
	// Return span ID as string
	return spanContext.SpanID().String()
}

// createResource creates a resource with service information
func createResource(serviceName, serviceVersion, environment string) *resource.Resource {
	// Create a resource with service name, version, and environment
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(serviceVersion),
		attribute.String("environment", environment),
		// Add hostname
		attribute.String("host.name", getHostname()),
	)
}

// getHostname returns the hostname or "unknown" if it cannot be determined
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// createExporter creates a span exporter based on configuration
func createExporter(config TracingConfig) (sdktrace.SpanExporter, error) {
	switch config.ExporterType {
	case "otlp":
		// Create OTLP exporter for OpenTelemetry Collector
		return otlptrace.New(
			context.Background(),
			otlptracegrpc.NewClient(
				otlptracegrpc.WithEndpoint(config.Endpoint),
				otlptracegrpc.WithInsecure(), // For development; use TLS in production
			),
		)
	case "jaeger":
		// Create Jaeger exporter
		return jaeger.New(
			jaeger.WithCollectorEndpoint(
				jaeger.WithEndpoint(config.Endpoint),
			),
		)
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", config.ExporterType)
	}
}