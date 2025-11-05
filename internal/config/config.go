package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration parameters
type Config struct {
	OpenAIApiKey      string
	WeatherApiKey     string
	HolidayCalendarLink string
	RedisAddr         string
	MongoURI          string
	TelegramBotToken  string
	TelegramChatID    string
}

// Load loads configuration from environment variables and .env file
func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	config := &Config{
		OpenAIApiKey:      getEnv("OPENAI_API_KEY", ""),
		WeatherApiKey:     getEnv("WEATHER_API_KEY", ""),
		HolidayCalendarLink: getEnv("HOLIDAY_CALENDAR_LINK", "https://www.officeholidays.com/ics/spain/catalonia"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		MongoURI:          getEnv("MONGO_URI", "mongodb://acai:travel@localhost:27017"),
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    getEnv("TELEGRAM_CHAT_ID", ""),
	}

	// Validate required configuration
	if config.OpenAIApiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
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
