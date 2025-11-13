package logging

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/8adimka/Go_AI_Assistant/internal/logging"
)

func TestSecureLogger_Redaction(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buf, nil))
	secureLogger := logging.NewSecureLogger(baseLogger)

	tests := []struct {
		name     string
		args     []any
		expected string
	}{
		{
			name:     "redacts openai_api_key",
			args:     []any{"openai_api_key", "sk-1234567890abcdef"},
			expected: `"openai_api_key":"[REDACTED]"`,
		},
		{
			name:     "redacts weather_api_key",
			args:     []any{"weather_api_key", "abc123def456"},
			expected: `"weather_api_key":"[REDACTED]"`,
		},
		{
			name:     "redacts telegram_bot_token",
			args:     []any{"telegram_bot_token", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"},
			expected: `"telegram_bot_token":"[REDACTED]"`,
		},
		{
			name:     "redacts api_key",
			args:     []any{"api_key", "secret-key-123"},
			expected: `"api_key":"[REDACTED]"`,
		},
		{
			name:     "redacts password",
			args:     []any{"password", "mypassword123"},
			expected: `"password":"[REDACTED]"`,
		},
		{
			name:     "redacts token",
			args:     []any{"token", "jwt-token-here"},
			expected: `"token":"[REDACTED]"`,
		},
		{
			name:     "redacts secret",
			args:     []any{"secret", "very-secret-value"},
			expected: `"secret":"[REDACTED]"`,
		},
		{
			name:     "preserves non-sensitive fields",
			args:     []any{"user_id", "12345", "message", "hello world"},
			expected: `"user_id":"12345","message":"hello world"`,
		},
		{
			name:     "handles mixed sensitive and non-sensitive fields",
			args:     []any{"user_id", "12345", "api_key", "secret-key", "message", "test"},
			expected: `"user_id":"12345","api_key":"[REDACTED]","message":"test"`,
		},
		{
			name:     "handles case insensitive matching",
			args:     []any{"OpenAI_API_Key", "sk-1234567890abcdef"},
			expected: `"OpenAI_API_Key":"[REDACTED]"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			secureLogger.Info("test message", tt.args...)
			logOutput := buf.String()

			if !bytes.Contains([]byte(logOutput), []byte(tt.expected)) {
				t.Errorf("Expected log to contain %q, got: %s", tt.expected, logOutput)
			}

			// Ensure sensitive data is not present in the log
			for i := 0; i < len(tt.args); i += 2 {
				if key, ok := tt.args[i].(string); ok {
					// Check if this is a sensitive key by looking for redaction in output
					if bytes.Contains([]byte(logOutput), []byte(`"`+key+`":"[REDACTED]"`)) {
						if sensitiveValue, ok := tt.args[i+1].(string); ok {
							if bytes.Contains([]byte(logOutput), []byte(sensitiveValue)) {
								t.Errorf("Sensitive data %q found in log output: %s", sensitiveValue, logOutput)
							}
						}
					}
				}
			}
		})
	}
}

func TestSecureLogger_EdgeCases(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buf, nil))
	secureLogger := logging.NewSecureLogger(baseLogger)

	t.Run("handles odd number of arguments", func(t *testing.T) {
		buf.Reset()
		secureLogger.Info("test", "key1", "value1", "key2") // Odd number of args
		logOutput := buf.String()

		// Should not panic and should log what it can
		if logOutput == "" {
			t.Error("Expected some log output, got empty")
		}
	})

	t.Run("handles non-string keys", func(t *testing.T) {
		buf.Reset()
		secureLogger.Info("test", 123, "value", "key", "value2") // Non-string key
		logOutput := buf.String()

		// Should not panic
		if logOutput == "" {
			t.Error("Expected some log output, got empty")
		}
	})

	t.Run("handles empty arguments", func(t *testing.T) {
		buf.Reset()
		secureLogger.Info("test")
		logOutput := buf.String()

		if logOutput == "" {
			t.Error("Expected log output for empty args, got empty")
		}
	})
}

func TestSecureLogger_AllLogLevels(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	secureLogger := logging.NewSecureLogger(baseLogger)

	levels := []struct {
		name string
		log  func(msg string, args ...any)
	}{
		{"Info", secureLogger.Info},
		{"Error", secureLogger.Error},
		{"Warn", secureLogger.Warn},
		{"Debug", secureLogger.Debug},
	}

	for _, level := range levels {
		t.Run(level.name, func(t *testing.T) {
			buf.Reset()
			level.log("test message", "api_key", "secret-value", "user_id", "12345")
			logOutput := buf.String()

			if !bytes.Contains([]byte(logOutput), []byte(`"api_key":"[REDACTED]"`)) {
				t.Errorf("%s: Expected redaction of api_key, got: %s", level.name, logOutput)
			}

			if !bytes.Contains([]byte(logOutput), []byte(`"user_id":"12345"`)) {
				t.Errorf("%s: Expected preservation of user_id, got: %s", level.name, logOutput)
			}

			if bytes.Contains([]byte(logOutput), []byte("secret-value")) {
				t.Errorf("%s: Sensitive data found in log: %s", level.name, logOutput)
			}
		})
	}
}
