package tracing

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware creates HTTP middleware for tracing requests
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from headers
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Create span name from method and path
			spanName := r.Method + " " + r.URL.Path

			// Start span
			ctx, span := otel.Tracer(serviceName).Start(ctx, spanName,
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.host", r.Host),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("http.user_agent", r.UserAgent()),
					attribute.String("http.remote_addr", r.RemoteAddr),
					attribute.String("service.name", serviceName),
				),
			)
			defer span.End()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

			// Continue with request
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Add response attributes
			span.SetAttributes(
				attribute.Int("http.status_code", wrapped.statusCode),
				attribute.Int("http.response_size", wrapped.size),
			)

			// Mark span as error if status code is 5xx
			if wrapped.statusCode >= 500 {
				span.SetAttributes(attribute.Bool("error", true))
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// ClientTracingMiddleware creates HTTP client middleware for tracing outgoing requests
func ClientTracingMiddleware(serviceName string) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return &roundTripper{
			next:        next,
			serviceName: serviceName,
		}
	}
}

// roundTripper implements http.RoundTripper with tracing
type roundTripper struct {
	next        http.RoundTripper
	serviceName string
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Start span for outgoing request
	ctx, span := otel.Tracer(rt.serviceName).Start(req.Context(), "HTTP "+req.Method,
		trace.WithAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL.String()),
			attribute.String("http.host", req.Host),
			attribute.String("service.name", rt.serviceName),
		),
	)
	defer span.End()

	// Inject trace context into headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// Execute request
	resp, err := rt.next.RoundTrip(req.WithContext(ctx))

	// Add response attributes
	if err != nil {
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)
	} else if resp != nil {
		span.SetAttributes(
			attribute.Int("http.status_code", resp.StatusCode),
		)

		// Mark span as error if status code is 5xx
		if resp.StatusCode >= 500 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}

	return resp, err
}
