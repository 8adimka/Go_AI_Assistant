package logging

import (
	"log/slog"
	"strings"
)

// SecureLogger provides logging with sensitive data redaction
type SecureLogger struct {
	logger         *slog.Logger
	redactedFields []string
}

// NewSecureLogger creates a new secure logger
func NewSecureLogger(logger *slog.Logger) *SecureLogger {
	return &SecureLogger{
		logger: logger,
		redactedFields: []string{
			"openai_api_key",
			"weather_api_key",
			"telegram_bot_token",
			"api_key",
			"password",
			"token",
			"secret",
			"key",
		},
	}
}

// redactSensitive filters out sensitive fields from log arguments
func (sl *SecureLogger) redactSensitive(args []any) []any {
	if len(args) == 0 {
		return args
	}

	// Handle key-value pairs
	if len(args)%2 != 0 {
		return args // Not key-value pairs, return as-is
	}

	result := make([]any, len(args))
	copy(result, args)

	for i := 0; i < len(result); i += 2 {
		key, ok := result[i].(string)
		if !ok {
			continue
		}

		// Check if this field should be redacted
		if sl.shouldRedact(key) {
			result[i+1] = "[REDACTED]"
		}
	}

	return result
}

// shouldRedact checks if a field name indicates sensitive data
func (sl *SecureLogger) shouldRedact(fieldName string) bool {
	fieldName = strings.ToLower(fieldName)

	for _, redacted := range sl.redactedFields {
		if strings.Contains(fieldName, redacted) {
			return true
		}
	}
	return false
}

// Info logs at Info level with sensitive data redaction
func (sl *SecureLogger) Info(msg string, args ...any) {
	sl.logger.Info(msg, sl.redactSensitive(args)...)
}

// Error logs at Error level with sensitive data redaction
func (sl *SecureLogger) Error(msg string, args ...any) {
	sl.logger.Error(msg, sl.redactSensitive(args)...)
}

// Warn logs at Warn level with sensitive data redaction
func (sl *SecureLogger) Warn(msg string, args ...any) {
	sl.logger.Warn(msg, sl.redactSensitive(args)...)
}

// Debug logs at Debug level with sensitive data redaction
func (sl *SecureLogger) Debug(msg string, args ...any) {
	sl.logger.Debug(msg, sl.redactSensitive(args)...)
}
