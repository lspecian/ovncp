package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	Enabled      bool
	ServiceName  string
	Environment  string
	Version      string
	ExporterType string // "jaeger" or "otlp"
	Endpoint     string
	SampleRate   float64
}

// TracerProvider wraps the OpenTelemetry tracer provider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// InitTracing initializes the tracing system
func InitTracing(cfg Config) (*TracerProvider, error) {
	if !cfg.Enabled {
		// Return no-op tracer if tracing is disabled
		return &TracerProvider{
			tracer: otel.Tracer(cfg.ServiceName),
		}, nil
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.Version),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter
	var exporter sdktrace.SpanExporter
	switch cfg.ExporterType {
	case "jaeger":
		exporter, err = createJaegerExporter(cfg.Endpoint)
	case "otlp":
		exporter, err = createOTLPExporter(cfg.Endpoint)
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", cfg.ExporterType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	// Register as global provider
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &TracerProvider{
		provider: provider,
		tracer:   provider.Tracer(cfg.ServiceName),
	}, nil
}

// createJaegerExporter creates a Jaeger exporter
func createJaegerExporter(endpoint string) (sdktrace.SpanExporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
}

// createOTLPExporter creates an OTLP exporter
func createOTLPExporter(endpoint string) (sdktrace.SpanExporter, error) {
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	return otlptrace.New(context.Background(), client)
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider != nil {
		return tp.provider.Shutdown(ctx)
	}
	return nil
}

// Tracer returns the tracer instance
func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("ovncp")
	return tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordSpanError records an error on the current span
func RecordSpanError(ctx context.Context, err error) {
	span := SpanFromContext(ctx)
	span.RecordError(err)
}

// Common attribute keys for OVN operations
var (
	ResourceTypeKey     = attribute.Key("ovn.resource.type")
	ResourceIDKey       = attribute.Key("ovn.resource.id")
	ResourceNameKey     = attribute.Key("ovn.resource.name")
	OperationTypeKey    = attribute.Key("ovn.operation.type")
	UserIDKey           = attribute.Key("user.id")
	UserEmailKey        = attribute.Key("user.email")
	TransactionIDKey    = attribute.Key("ovn.transaction.id")
	TransactionOpsKey   = attribute.Key("ovn.transaction.operations")
	DBQueryKey          = attribute.Key("db.query")
	DBTableKey          = attribute.Key("db.table")
	HTTPMethodKey       = attribute.Key("http.method")
	HTTPPathKey         = attribute.Key("http.path")
	HTTPStatusCodeKey   = attribute.Key("http.status_code")
	ErrorMessageKey     = attribute.Key("error.message")
	ErrorTypeKey        = attribute.Key("error.type")
)

// Helper functions for common span attributes
func WithResource(resourceType, resourceID, resourceName string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		ResourceTypeKey.String(resourceType),
	}
	if resourceID != "" {
		attrs = append(attrs, ResourceIDKey.String(resourceID))
	}
	if resourceName != "" {
		attrs = append(attrs, ResourceNameKey.String(resourceName))
	}
	return attrs
}

func WithOperation(operationType string) attribute.KeyValue {
	return OperationTypeKey.String(operationType)
}

func WithUser(userID, userEmail string) []attribute.KeyValue {
	return []attribute.KeyValue{
		UserIDKey.String(userID),
		UserEmailKey.String(userEmail),
	}
}

func WithTransaction(transactionID string, operationCount int) []attribute.KeyValue {
	return []attribute.KeyValue{
		TransactionIDKey.String(transactionID),
		TransactionOpsKey.Int(operationCount),
	}
}

func WithDatabase(query, table string) []attribute.KeyValue {
	return []attribute.KeyValue{
		DBQueryKey.String(query),
		DBTableKey.String(table),
	}
}

func WithHTTP(method, path string, statusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{
		HTTPMethodKey.String(method),
		HTTPPathKey.String(path),
		HTTPStatusCodeKey.Int(statusCode),
	}
}

func WithError(err error, errorType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		ErrorMessageKey.String(err.Error()),
		ErrorTypeKey.String(errorType),
	}
}