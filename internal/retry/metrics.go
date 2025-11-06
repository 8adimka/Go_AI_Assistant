package retry

import (
	"context"
)

// Metrics holds retry-related metrics (stub implementation)
type Metrics struct{}

// NewMetrics creates new retry metrics (stub implementation)
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordAttempt records a retry attempt (stub implementation)
func (m *Metrics) RecordAttempt(service string) {
	// Stub implementation - metrics will be added later
}

// RecordFailure records a retry failure (stub implementation)
func (m *Metrics) RecordFailure(service string) {
	// Stub implementation - metrics will be added later
}

// RecordSuccess records a successful retry (stub implementation)
func (m *Metrics) RecordSuccess(service string) {
	// Stub implementation - metrics will be added later
}

// WithMetrics wraps a retry function with metrics collection (stub implementation)
func WithMetrics(ctx context.Context, config RetryConfig, service string, fn RetryableFunc) error {
	// For now, just use the regular retry logic without metrics
	return Retry(ctx, config, fn)
}

// WithMetricsWithResult wraps a retry function with metrics collection and result (stub implementation)
func WithMetricsWithResult[T any](ctx context.Context, config RetryConfig, service string, fn func() (T, error)) (T, error) {
	// For now, just use the regular retry logic without metrics
	return RetryWithResult(ctx, config, fn)
}
