package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration parameters
type Config struct {
	OpenAIApiKey        string
	OpenAIModel         string
	WeatherApiKey       string
	HolidayCalendarLink string
	RedisAddr           string
	MongoURI            string
	TelegramBotToken    string
	TelegramChatID      string
	RetryMaxAttempts    int
	RetryBaseDelayMs    int
	RetryMaxDelayMs     int

	// API Security
	APIKey string // API key for protecting sensitive endpoints

	// Rate Limiting
	APIRateLimitRPS   float64 // Requests per second
	APIRateLimitBurst int     // Burst size

	// Cache TTL
	CacheTTLHours     int // Redis cache TTL in hours
	SessionTTLMinutes int // Session TTL in minutes

	// Circuit Breaker
	CircuitBreakerMaxFailures     int // Max failures before opening circuit
	CircuitBreakerCooldownSeconds int // Cooldown period in seconds
}

// Load loads configuration from environment variables and .env file
func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	config := &Config{
		OpenAIApiKey:        getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:         getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		WeatherApiKey:       getEnv("WEATHER_API_KEY", ""),
		HolidayCalendarLink: getEnv("HOLIDAY_CALENDAR_LINK", "https://www.officeholidays.com/ics/spain/catalonia"),
		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		MongoURI:            getEnv("MONGO_URI", "mongodb://acai:travel@localhost:27017"),
		TelegramBotToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:      getEnv("TELEGRAM_CHAT_ID", ""),
		RetryMaxAttempts:    getEnvInt("RETRY_MAX_ATTEMPTS", 3),
		RetryBaseDelayMs:    getEnvInt("RETRY_BASE_DELAY_MS", 500),
		RetryMaxDelayMs:     getEnvInt("RETRY_MAX_DELAY_MS", 5000),

		// API Security
		APIKey: getEnv("API_KEY", ""),

		// Rate Limiting
		APIRateLimitRPS:   getEnvFloat("API_RATE_LIMIT_RPS", 10.0),
		APIRateLimitBurst: getEnvInt("API_RATE_LIMIT_BURST", 20),

		// Cache TTL
		CacheTTLHours:     getEnvInt("CACHE_TTL_HOURS", 24),
		SessionTTLMinutes: getEnvInt("SESSION_TTL_MINUTES", 30),

		// Circuit Breaker
		CircuitBreakerMaxFailures:     getEnvInt("CIRCUIT_BREAKER_MAX_FAILURES", 3),
		CircuitBreakerCooldownSeconds: getEnvInt("CIRCUIT_BREAKER_COOLDOWN_SECONDS", 30),
	}

	// Validate required configuration
	if config.OpenAIApiKey == "" {
		log.Printf("Warning: OPENAI_API_KEY is required for production use")
		// Don't fatal in tests to allow them to run
	}

	return config
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

// getEnvInt gets environment variable as integer with fallback
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
		log.Printf("Warning: invalid integer value for %s: %s, using default: %d", key, value, fallback)
	}
	return fallback
}

// getEnvFloat gets environment variable as float64 with fallback
func getEnvFloat(key string, fallback float64) float64 {
	if value := os.Getenv(key); value != "" {
		var result float64
		if _, err := fmt.Sscanf(value, "%f", &result); err == nil {
			return result
		}
		log.Printf("Warning: invalid float value for %s: %s, using default: %f", key, value, fallback)
	}
	return fallback
}
