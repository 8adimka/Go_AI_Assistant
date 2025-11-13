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
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	twirpRequestsTotal  metric.Int64Counter

	// Simplified OpenAI metrics
	openaiRequestsTotal   metric.Int64Counter
	openaiRequestDuration metric.Float64Histogram

	// Token usage metrics
	tokenUsageTotal      metric.Int64Counter
	tokenUsageByModel    metric.Int64Counter
	contextTokenCount    metric.Int64Histogram
	tokenEstimationError metric.Float64Histogram
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

	twirpRequestsTotal, err := meter.Int64Counter(
		"twirp_requests_total",
		metric.WithDescription("Total number of Twirp requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Simplified OpenAI metrics
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

	// Token usage metrics
	tokenUsageTotal, err := meter.Int64Counter(
		"token_usage_total",
		metric.WithDescription("Total tokens used across all operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	tokenUsageByModel, err := meter.Int64Counter(
		"token_usage_by_model",
		metric.WithDescription("Token usage broken down by model"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	contextTokenCount, err := meter.Int64Histogram(
		"context_token_count",
		metric.WithDescription("Distribution of context token counts"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(100, 500, 1000, 2000, 4000, 8000, 16000, 32000, 64000, 128000),
	)
	if err != nil {
		return nil, err
	}

	tokenEstimationError, err := meter.Float64Histogram(
		"token_estimation_error_percent",
		metric.WithDescription("Percentage error in token estimation"),
		metric.WithUnit("%"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 20, 50, 100),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		httpRequestsTotal:     httpRequestsTotal,
		httpRequestDuration:   httpRequestDuration,
		twirpRequestsTotal:    twirpRequestsTotal,
		openaiRequestsTotal:   openaiRequestsTotal,
		openaiRequestDuration: openaiRequestDuration,
		tokenUsageTotal:       tokenUsageTotal,
		tokenUsageByModel:     tokenUsageByModel,
		contextTokenCount:     contextTokenCount,
		tokenEstimationError:  tokenEstimationError,
	}, nil
}

// HTTPMetricsMiddleware returns middleware for collecting HTTP metrics
func (m *Metrics) HTTPMetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

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

// RecordOpenAIRequest records simplified OpenAI request metrics
func (m *Metrics) RecordOpenAIRequest(ctx context.Context, operation, model, userID, platform string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation), // "title" or "reply"
		attribute.String("model", model),
		attribute.String("user_id", userID),
		attribute.String("platform", platform),
	}

	m.openaiRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.openaiRequestDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
}

// RecordTokenUsage records token usage metrics
func (m *Metrics) RecordTokenUsage(ctx context.Context, operation, model string, promptTokens, completionTokens, totalTokens int64) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("model", model),
	}

	// Record total tokens
	m.tokenUsageTotal.Add(ctx, totalTokens, metric.WithAttributes(attrs...))

	// Record breakdown by model
	modelAttrs := []attribute.KeyValue{
		attribute.String("model", model),
		attribute.String("token_type", "prompt"),
	}
	m.tokenUsageByModel.Add(ctx, promptTokens, metric.WithAttributes(modelAttrs...))

	modelAttrs = []attribute.KeyValue{
		attribute.String("model", model),
		attribute.String("token_type", "completion"),
	}
	m.tokenUsageByModel.Add(ctx, completionTokens, metric.WithAttributes(modelAttrs...))

	modelAttrs = []attribute.KeyValue{
		attribute.String("model", model),
		attribute.String("token_type", "total"),
	}
	m.tokenUsageByModel.Add(ctx, totalTokens, metric.WithAttributes(modelAttrs...))
}

// RecordContextTokenCount records the size of conversation contexts
func (m *Metrics) RecordContextTokenCount(ctx context.Context, conversationID, platform string, tokenCount int64) {
	attrs := []attribute.KeyValue{
		attribute.String("conversation_id", conversationID),
		attribute.String("platform", platform),
	}
	m.contextTokenCount.Record(ctx, tokenCount, metric.WithAttributes(attrs...))
}

// RecordTokenEstimationError records the accuracy of token estimation
func (m *Metrics) RecordTokenEstimationError(ctx context.Context, operation string, estimatedTokens, actualTokens int) {
	if actualTokens == 0 {
		return // Avoid division by zero
	}

	errorPercent := float64(abs(estimatedTokens-actualTokens)) / float64(actualTokens) * 100

	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
	}
	m.tokenEstimationError.Record(ctx, errorPercent, metric.WithAttributes(attrs...))
}

// RecordOpenAIRequestWithTokens records OpenAI request with detailed token metrics
func (m *Metrics) RecordOpenAIRequestWithTokens(ctx context.Context, operation, model, userID, platform string, duration time.Duration, promptTokens, completionTokens, totalTokens int64) {
	// Record basic OpenAI metrics
	m.RecordOpenAIRequest(ctx, operation, model, userID, platform, duration)

	// Record detailed token usage
	m.RecordTokenUsage(ctx, operation, model, promptTokens, completionTokens, totalTokens)
}

// Helper function for absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
