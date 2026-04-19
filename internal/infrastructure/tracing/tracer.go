// Package tracing provides OpenTelemetry setup and utilities for the betting platform.
//
// It includes:
// - Tracer provider initialization with Jaeger exporter
// - HTTP middleware for automatic request tracing
// - Database tracing for PostgreSQL operations
// - Event bus tracing for NATS operations
// - Context propagation utilities
package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	JaegerEndpoint string
	OTLPEndpoint   string
	SamplingRatio  float64 // 0.0 to 1.0
	Enabled        bool
}

// DefaultConfig returns a default tracing configuration
func DefaultConfig(serviceName string) Config {
	return Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    os.Getenv("ENVIRONMENT"),
		JaegerEndpoint: os.Getenv("JAEGER_ENDPOINT"),
		OTLPEndpoint:   os.Getenv("OTLP_ENDPOINT"),
		SamplingRatio:  1.0, // Sample all traces in development
		Enabled:        os.Getenv("OTEL_ENABLED") != "false",
	}
}

// InitTracer initializes the OpenTelemetry tracer provider
func InitTracer(ctx context.Context, cfg Config) (func(), error) {
	if !cfg.Enabled {
		return func() {}, nil
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on configuration
	var exporter sdktrace.SpanExporter
	if cfg.OTLPEndpoint != "" {
		// Use OTLP exporter (for systems like Tempo, Honeycomb, etc.)
		exporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		)
	} else if cfg.JaegerEndpoint != "" {
		// Use Jaeger exporter
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.JaegerEndpoint)))
	} else {
		// Default to Jaeger on localhost
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRatio)),
	)

	// Register as global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Return cleanup function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			fmt.Printf("Error shutting down tracer provider: %v\n", err)
		}
	}, nil
}

// Tracer returns a tracer for the service
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer("betting-platform").Start(ctx, name, opts...)
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetSpanError marks the current span as having an error
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attribute.String("error.message", err.Error()))
		span.RecordError(err)
	}
}

// WithSpan creates a span around the execution of a function
func WithSpan(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, name)
	defer span.End()

	if err := fn(ctx); err != nil {
		SetSpanError(ctx, err)
		return err
	}

	return nil
}

// WithSpanValue creates a span around a function that returns a value
func WithSpanValue[T any](ctx context.Context, name string, fn func(context.Context) (T, error)) (T, error) {
	ctx, span := StartSpan(ctx, name)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		SetSpanError(ctx, err)
		return result, err
	}

	return result, nil
}

// Common span attributes
const (
	AttrUserID      = attribute.Key("user.id")
	AttrBetID       = attribute.Key("bet.id")
	AttrGameID      = attribute.Key("game.id")
	AttrAmount      = attribute.Key("amount")
	AttrCurrency    = attribute.Key("currency")
	AttrCountryCode = attribute.Key("country.code")
	AttrPhoneNumber = attribute.Key("phone.number")
	AttrErrorCode   = attribute.Key("error.code")
	AttrProvider    = attribute.Key("provider")
	AttrMethod      = attribute.Key("method")
	AttrEndpoint    = attribute.Key("endpoint")
	AttrStatusCode  = attribute.Key("status.code")
)
