package errorsx

import (
	"errors"
	"fmt"

	"github.com/twitchtv/twirp"
)

// Common error types for the application
var (
	ErrNotFound     = errors.New("resource not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInternal     = errors.New("internal error")
	ErrTimeout      = errors.New("operation timeout")
	ErrUnavailable  = errors.New("service unavailable")
)

// Wrap wraps an error with additional context message
// Returns nil if the error is nil
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with a formatted context message
// Returns nil if the error is nil
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", message, err)
}

// ToTwirpError converts an internal error to a Twirp error with appropriate code
// It preserves the original error message and adds metadata for debugging
func ToTwirpError(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a Twirp error, return as is
	if _, ok := err.(twirp.Error); ok {
		return err
	}

	// Map internal errors to Twirp error codes
	switch {
	case errors.Is(err, ErrNotFound):
		return twirp.NotFoundError(err.Error())
	case errors.Is(err, ErrInvalidInput):
		return twirp.InvalidArgumentError("input", err.Error())
	case errors.Is(err, ErrUnauthorized):
		return twirp.NewError(twirp.Unauthenticated, err.Error())
	case errors.Is(err, ErrTimeout):
		return twirp.NewError(twirp.DeadlineExceeded, err.Error())
	case errors.Is(err, ErrUnavailable):
		return twirp.NewError(twirp.Unavailable, err.Error())
	default:
		// For unknown errors, return internal error
		return twirp.InternalErrorWith(err)
	}
}

// ToTwirpErrorWithMeta converts an error to Twirp error and adds metadata
func ToTwirpErrorWithMeta(err error, meta map[string]string) error {
	if err == nil {
		return nil
	}

	twirpErr := ToTwirpError(err)

	// Add metadata if it's a Twirp error
	if te, ok := twirpErr.(twirp.Error); ok {
		for key, value := range meta {
			te = te.WithMeta(key, value)
		}
		return te
	}

	return twirpErr
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsInvalidInput checks if an error is an invalid input error
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsUnauthorized checks if an error is an unauthorized error
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsTimeout checks if an error is a timeout error
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsUnavailable checks if an error is an unavailable error
func IsUnavailable(err error) bool {
	return errors.Is(err, ErrUnavailable)
}
