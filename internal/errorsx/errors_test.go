package errorsx

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/twitchtv/twirp"
)

func TestWrap(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
		want    string
	}{
		{
			name:    "wrap error with message",
			err:     errors.New("original error"),
			message: "context message",
			want:    "context message: original error",
		},
		{
			name:    "wrap nil error returns nil",
			err:     nil,
			message: "context message",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Wrap(tt.err, tt.message)
			if tt.want == "" && result != nil {
				t.Errorf("Expected nil, got %v", result)
			} else if tt.want != "" && result.Error() != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, result.Error())
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	err := errors.New("original error")
	result := Wrapf(err, "user %s failed operation %d", "john", 42)
	expected := "user john failed operation 42: original error"
	if result.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, result.Error())
	}

	// Test nil error
	result = Wrapf(nil, "should not wrap")
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestToTwirpError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode twirp.ErrorCode
		checkMessage bool
	}{
		{
			name:         "nil error returns nil",
			err:          nil,
			expectedCode: twirp.NoError,
		},
		{
			name:         "NotFound error maps to NotFound",
			err:          ErrNotFound,
			expectedCode: twirp.NotFound,
			checkMessage: true,
		},
		{
			name:         "wrapped NotFound error maps to NotFound",
			err:          fmt.Errorf("conversation not found: %w", ErrNotFound),
			expectedCode: twirp.NotFound,
		},
		{
			name:         "InvalidInput error maps to InvalidArgument",
			err:          ErrInvalidInput,
			expectedCode: twirp.InvalidArgument,
		},
		{
			name:         "Unauthorized error maps to Unauthenticated",
			err:          ErrUnauthorized,
			expectedCode: twirp.Unauthenticated,
		},
		{
			name:         "Timeout error maps to DeadlineExceeded",
			err:          ErrTimeout,
			expectedCode: twirp.DeadlineExceeded,
		},
		{
			name:         "Unavailable error maps to Unavailable",
			err:          ErrUnavailable,
			expectedCode: twirp.Unavailable,
		},
		{
			name:         "unknown error maps to Internal",
			err:          errors.New("random error"),
			expectedCode: twirp.Internal,
		},
		{
			name:         "existing Twirp error is preserved",
			err:          twirp.NotFoundError("already a twirp error"),
			expectedCode: twirp.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToTwirpError(tt.err)

			if tt.err == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			var code twirp.ErrorCode
			if te, ok := result.(twirp.Error); ok {
				code = te.Code()
			}
			if code != tt.expectedCode {
				t.Errorf("Expected code %v, got %v", tt.expectedCode, code)
			}

			// Verify error message is preserved
			if tt.checkMessage && !strings.Contains(result.Error(), tt.err.Error()) {
				t.Errorf("Expected message to contain %q, got %q", tt.err.Error(), result.Error())
			}
		})
	}
}

func TestToTwirpErrorWithMeta(t *testing.T) {
	err := ErrNotFound
	meta := map[string]string{
		"trace_id":        "abc123",
		"conversation_id": "conv456",
		"user_id":         "user789",
	}

	result := ToTwirpErrorWithMeta(err, meta)

	// Verify it's a Twirp error
	twirpErr, ok := result.(twirp.Error)
	if !ok {
		t.Fatal("Result is not a Twirp error")
	}

	// Verify error code
	if twirpErr.Code() != twirp.NotFound {
		t.Errorf("Expected NotFound code, got %v", twirpErr.Code())
	}

	// Verify metadata
	for key, expectedValue := range meta {
		actualValue := twirpErr.Meta(key)
		if actualValue != expectedValue {
			t.Errorf("Meta %q: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Test with nil error
	result = ToTwirpErrorWithMeta(nil, meta)
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "direct ErrNotFound",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "wrapped ErrNotFound",
			err:  fmt.Errorf("conversation not found: %w", ErrNotFound),
			want: true,
		},
		{
			name: "other error",
			err:  ErrInvalidInput,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInvalidInput(t *testing.T) {
	if !IsInvalidInput(ErrInvalidInput) {
		t.Error("Expected true for ErrInvalidInput")
	}

	wrapped := fmt.Errorf("validation failed: %w", ErrInvalidInput)
	if !IsInvalidInput(wrapped) {
		t.Error("Expected true for wrapped ErrInvalidInput")
	}

	if IsInvalidInput(ErrNotFound) {
		t.Error("Expected false for different error")
	}

	if IsInvalidInput(nil) {
		t.Error("Expected false for nil")
	}
}

func TestIsUnauthorized(t *testing.T) {
	if !IsUnauthorized(ErrUnauthorized) {
		t.Error("Expected true for ErrUnauthorized")
	}

	if IsUnauthorized(ErrInvalidInput) {
		t.Error("Expected false for different error")
	}
}

func TestIsTimeout(t *testing.T) {
	if !IsTimeout(ErrTimeout) {
		t.Error("Expected true for ErrTimeout")
	}

	if IsTimeout(ErrInvalidInput) {
		t.Error("Expected false for different error")
	}
}

func TestIsUnavailable(t *testing.T) {
	if !IsUnavailable(ErrUnavailable) {
		t.Error("Expected true for ErrUnavailable")
	}

	if IsUnavailable(ErrInvalidInput) {
		t.Error("Expected false for different error")
	}
}

func TestErrorChaining(t *testing.T) {
	// Test error chaining works correctly
	baseErr := errors.New("database connection failed")
	wrappedOnce := Wrap(baseErr, "failed to query user")
	wrappedTwice := Wrap(wrappedOnce, "GetUser operation failed")

	// Should contain all messages
	errMsg := wrappedTwice.Error()
	if !strings.Contains(errMsg, "GetUser operation failed") {
		t.Error("Should contain outer message")
	}
	if !strings.Contains(errMsg, "failed to query user") {
		t.Error("Should contain middle message")
	}
	if !strings.Contains(errMsg, "database connection failed") {
		t.Error("Should contain inner message")
	}

	// Should unwrap to original error
	if !errors.Is(wrappedTwice, baseErr) {
		t.Error("Should be able to unwrap to original error")
	}
}

func TestToTwirpErrorPreservesStack(t *testing.T) {
	// Create a chain of wrapped errors
	baseErr := errors.New("database error")
	wrapped := Wrap(baseErr, "query failed")
	wrappedAgain := Wrap(wrapped, "operation failed")

	// Convert to Twirp error
	twirpErr := ToTwirpError(wrappedAgain)

	// Message should preserve the chain
	msg := twirpErr.Error()
	if !strings.Contains(msg, "operation failed") {
		t.Errorf("Expected message to contain 'operation failed', got: %s", msg)
	}
}

func TestMultipleMeta(t *testing.T) {
	err := ErrTimeout
	meta1 := map[string]string{
		"request_id": "req123",
		"user_id":    "user456",
	}
	meta2 := map[string]string{
		"trace_id": "trace789",
		"span_id":  "span000",
	}

	// Add first set of metadata
	result := ToTwirpErrorWithMeta(err, meta1)
	twirpErr1, _ := result.(twirp.Error)

	// Add second set of metadata
	result = ToTwirpErrorWithMeta(result, meta2)
	twirpErr2, _ := result.(twirp.Error)

	// Verify all metadata is present
	allMeta := make(map[string]string)
	for k, v := range meta1 {
		allMeta[k] = v
	}
	for k, v := range meta2 {
		allMeta[k] = v
	}

	for key, expectedValue := range allMeta {
		actualValue := twirpErr2.Meta(key)
		if actualValue != expectedValue {
			t.Errorf("Meta %q: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Verify first error still has its metadata
	if twirpErr1.Meta("request_id") != "req123" {
		t.Error("First error lost its metadata")
	}
}
