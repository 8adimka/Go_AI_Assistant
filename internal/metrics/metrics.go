package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all application metrics
type Metrics struct {
	httpRequestsTotal    metric.Int64Counter
	httpRequestDuration  metric.Float64Histogram
	httpRequestsInFlight metric.Int64UpDownCounter
	twirpRequestsTotal   metric.Int64Counter
}

// NewMetrics creates and initializes all metrics
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	httpRequestsTotal, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	httpRequestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	httpRequestsInFlight, err := meter.Int64UpDownCounter(
		"http_requests_in_progress",
		metric.WithDescription("Number of HTTP requests currently in progress"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	twirpRequestsTotal, err := meter.Int64Counter(
		"twirp_requests_total",
		metric.WithDescription("Total number of Twirp requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		httpRequestsTotal:    httpRequestsTotal,
		httpRequestDuration:  httpRequestDuration,
		httpRequestsInFlight: httpRequestsInFlight,
		twirpRequestsTotal:   twirpRequestsTotal,
	}, nil
}

// HTTPMetricsMiddleware returns middleware for collecting HTTP metrics
func (m *Metrics) HTTPMetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Increment in-flight requests
			m.httpRequestsInFlight.Add(r.Context(), 1)

			// Create response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(rw.statusCode)

			m.httpRequestsTotal.Add(r.Context(), 1,
				metric.WithAttributes(
					attribute.String("method", r.Method),
					attribute.String("path", r.URL.Path),
					attribute.String("status", statusCode),
				),
			)

			m.httpRequestDuration.Record(r.Context(), duration,
				metric.WithAttributes(
					attribute.String("method", r.Method),
					attribute.String("path", r.URL.Path),
					attribute.String("status", statusCode),
				),
			)

			// Decrement in-flight requests
			m.httpRequestsInFlight.Add(r.Context(), -1)
		})
	}
}

// RecordTwirpRequest records metrics for Twirp requests
func (m *Metrics) RecordTwirpRequest(ctx context.Context, method string, status string) {
	m.twirpRequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("status", status),
		),
	)
}

// responseWriter captures the status code for metrics
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
