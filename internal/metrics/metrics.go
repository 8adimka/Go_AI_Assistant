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

	// OpenAI metrics
	openaiTokensInput     metric.Int64Counter
	openaiTokensOutput    metric.Int64Counter
	openaiTokensTotal     metric.Int64Counter
	openaiRequestsTotal   metric.Int64Counter
	openaiRequestDuration metric.Float64Histogram
}

// NewMetrics creates and initializes all metrics
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	httpRequestsTotal, err := meter.Int64Counter(
		"http_server_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	httpRequestDuration, err := meter.Float64Histogram(
		"http_server_latency_ms",
		metric.WithDescription("HTTP request latency in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	httpRequestsInFlight, err := meter.Int64UpDownCounter(
		"http_server_requests_in_progress",
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

	// OpenAI metrics
	openaiTokensInput, err := meter.Int64Counter(
		"openai_tokens_input_total",
		metric.WithDescription("Total OpenAI input tokens consumed"),
		metric.WithUnit("tokens"),
	)
	if err != nil {
		return nil, err
	}

	openaiTokensOutput, err := meter.Int64Counter(
		"openai_tokens_output_total",
		metric.WithDescription("Total OpenAI output tokens consumed"),
		metric.WithUnit("tokens"),
	)
	if err != nil {
		return nil, err
	}

	openaiTokensTotal, err := meter.Int64Counter(
		"openai_tokens_total",
		metric.WithDescription("Total OpenAI tokens consumed"),
		metric.WithUnit("tokens"),
	)
	if err != nil {
		return nil, err
	}

	openaiRequestsTotal, err := meter.Int64Counter(
		"openai_requests_total",
		metric.WithDescription("Total OpenAI API requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	openaiRequestDuration, err := meter.Float64Histogram(
		"openai_request_duration_ms",
		metric.WithDescription("OpenAI API request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		httpRequestsTotal:     httpRequestsTotal,
		httpRequestDuration:   httpRequestDuration,
		httpRequestsInFlight:  httpRequestsInFlight,
		twirpRequestsTotal:    twirpRequestsTotal,
		openaiTokensInput:     openaiTokensInput,
		openaiTokensOutput:    openaiTokensOutput,
		openaiTokensTotal:     openaiTokensTotal,
		openaiRequestsTotal:   openaiRequestsTotal,
		openaiRequestDuration: openaiRequestDuration,
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
			durationMs := float64(time.Since(start).Nanoseconds()) / 1e6 // Convert to milliseconds
			statusCode := strconv.Itoa(rw.statusCode)

			m.httpRequestsTotal.Add(r.Context(), 1,
				metric.WithAttributes(
					attribute.String("method", r.Method),
					attribute.String("path", r.URL.Path),
					attribute.String("status_code", statusCode),
				),
			)

			m.httpRequestDuration.Record(r.Context(), durationMs,
				metric.WithAttributes(
					attribute.String("method", r.Method),
					attribute.String("path", r.URL.Path),
					attribute.String("status_code", statusCode),
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

// TokenUsage represents OpenAI token usage
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// RecordOpenAITokens records OpenAI token usage metrics
func (m *Metrics) RecordOpenAITokens(ctx context.Context, operation, model, userID, platform string, usage TokenUsage, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation), // "title" or "reply"
		attribute.String("model", model),
		attribute.String("user_id", userID),
		attribute.String("platform", platform),
	}

	m.openaiTokensInput.Add(ctx, int64(usage.PromptTokens), metric.WithAttributes(attrs...))
	m.openaiTokensOutput.Add(ctx, int64(usage.CompletionTokens), metric.WithAttributes(attrs...))
	m.openaiTokensTotal.Add(ctx, int64(usage.TotalTokens), metric.WithAttributes(attrs...))
	m.openaiRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.openaiRequestDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
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
