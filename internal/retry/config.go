package retry

import (
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/config"
)

// ConfigFromAppConfig creates a RetryConfig from the application configuration
func ConfigFromAppConfig(appConfig *config.Config) RetryConfig {
	return RetryConfig{
		MaxAttempts: appConfig.RetryMaxAttempts,
		BaseDelay:   time.Duration(appConfig.RetryBaseDelayMs) * time.Millisecond,
		MaxDelay:    time.Duration(appConfig.RetryMaxDelayMs) * time.Millisecond,
	}
}
