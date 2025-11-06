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
