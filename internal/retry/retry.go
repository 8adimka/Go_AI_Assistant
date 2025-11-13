package retry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxAttempts int           // Maximum number of retry attempts (default: 3)
	BaseDelay   time.Duration // Base delay between retries (default: 500ms)
	MaxDelay    time.Duration // Maximum delay between retries (default: 5s)
}

// DefaultConfig returns the default retry configuration
func DefaultConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    5 * time.Second,
	}
}

// RetryWithResult executes a function that returns a result with retry logic
func RetryWithResult[T any](ctx context.Context, config RetryConfig, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		result, err := fn()
		if err == nil {
			// Success - return immediately
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			slog.WarnContext(ctx, "Non-retryable error encountered, not retrying",
				"attempt", attempt+1,
				"error", err)
			return zero, err
		}

		// Check if we've reached max attempts
		if attempt == config.MaxAttempts {
			slog.WarnContext(ctx, "Max retry attempts reached, giving up",
				"attempts", config.MaxAttempts+1,
				"error", err)
			return zero, fmt.Errorf("max retry attempts (%d) reached, last error: %w", config.MaxAttempts+1, err)
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(config, attempt)
		slog.WarnContext(ctx, "Retryable error encountered, will retry",
			"attempt", attempt+1,
			"max_attempts", config.MaxAttempts+1,
			"delay", delay,
			"error", err)

		// Wait for the delay or context cancellation
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return zero, lastErr
}

// isRetryableError determines if an error should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for OpenAI API errors
	var openaiErr *openai.Error
	if errors.As(err, &openaiErr) {
		// Rate limits and server errors are retryable
		errorStr := openaiErr.Error()
		return strings.Contains(errorStr, "rate limit") ||
			strings.Contains(errorStr, "server") ||
			strings.Contains(errorStr, "timeout")
	}

	// Check for HTTP errors
	var httpErr interface {
		StatusCode() int
	}
	if errors.As(err, &httpErr) {
		statusCode := httpErr.StatusCode()
		// Retry on server errors (5xx) and rate limits
		return statusCode >= 500 || statusCode == http.StatusTooManyRequests
	}

	// Check for network/timeout errors
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) ||
		isNetworkError(err)
}

// isNetworkError checks if error is a network-related error
func isNetworkError(err error) bool {
	errorStr := strings.ToLower(err.Error())
	networkKeywords := []string{
		"connection",
		"timeout",
		"network",
		"dial",
		"EOF",
		"reset",
		"refused",
	}

	for _, keyword := range networkKeywords {
		if strings.Contains(errorStr, keyword) {
			return true
		}
	}
	return false
}

// calculateDelay computes the delay for exponential backoff
func calculateDelay(config RetryConfig, attempt int) time.Duration {
	// Simple exponential backoff: base * 2^attempt
	delay := config.BaseDelay * time.Duration(1<<uint(attempt))

	// Cap at max delay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	return delay
}
