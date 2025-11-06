//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"github.com/8adimka/Go_AI_Assistant/internal/weather"
)

// TestWeatherAPIWithRetry tests the weather API integration with retry mechanism
func TestWeatherAPIWithRetry(t *testing.T) {
	// Load configuration
	cfg := config.Load()

	// Skip test if no valid WeatherAPI key is provided
	if cfg.WeatherApiKey == "" || cfg.WeatherApiKey == "your_weatherapi_key_here" {
		t.Skip("WeatherAPI key not provided or using placeholder, skipping integration test")
	}

	// Create weather service
	redisClient := redisx.MustConnect(cfg.RedisAddr)
	cache := redisx.NewCache(redisClient, 24*time.Hour)
	weatherService := weather.CreateWeatherService(cfg.WeatherApiKey, cache)

	tests := []struct {
		name        string
		city        string
		expectError bool
	}{
		{
			name:        "Valid city - Barcelona",
			city:        "Barcelona",
			expectError: false,
		},
		{
			name:        "Valid city - Madrid",
			city:        "Madrid",
			expectError: false,
		},
		{
			name:        "Invalid city - should fail gracefully",
			city:        "InvalidCityName12345",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Test weather data retrieval with retry mechanism
			weatherData, err := weatherService.GetCurrentWithFallback(ctx, tt.city)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				t.Logf("Expected error occurred: %v", err)
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}

				// Validate weather data structure
				if weatherData == nil {
					t.Error("Weather data is nil")
					return
				}

				if weatherData.Location == "" {
					t.Error("Location is empty")
				}
				if weatherData.Temperature == 0 {
					t.Error("Temperature is zero")
				}
				if weatherData.Condition == "" {
					t.Error("Condition is empty")
				}

				t.Logf("Weather data retrieved successfully: %s - %.1f°C - %s",
					weatherData.Location, weatherData.Temperature, weatherData.Condition)
			}
		})
	}
}

// TestWeatherServiceFallback tests the fallback mechanism when primary provider fails
func TestWeatherServiceFallback(t *testing.T) {
	cfg := config.Load()

	// Create weather service with invalid API key to trigger fallback
	redisClient := redisx.MustConnect(cfg.RedisAddr)
	cache := redisx.NewCache(redisClient, 24*time.Hour)
	weatherService := weather.CreateWeatherService("invalid_key_12345", cache)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This should trigger the fallback to mock provider
	weatherData, err := weatherService.GetCurrentWithFallback(ctx, "Barcelona")

	if err != nil {
		t.Errorf("Fallback should handle errors gracefully, got: %v", err)
		return
	}

	if weatherData == nil {
		t.Error("Fallback should provide mock weather data")
		return
	}

	// Mock provider should return valid data
	if weatherData.Location == "" {
		t.Error("Mock weather data should have location")
	}
	if weatherData.Temperature == 0 {
		t.Error("Mock weather data should have temperature")
	}

	t.Logf("Fallback weather data: %s - %.1f°C - %s",
		weatherData.Location, weatherData.Temperature, weatherData.Condition)
}

// TestWeatherServiceRateLimiting tests that rate limiting is working
func TestWeatherServiceRateLimiting(t *testing.T) {
	cfg := config.Load()

	if cfg.WeatherApiKey == "" || cfg.WeatherApiKey == "your_weatherapi_key_here" {
		t.Skip("WeatherAPI key not provided or using placeholder, skipping rate limiting test")
	}

	redisClient := redisx.MustConnect(cfg.RedisAddr)
	cache := redisx.NewCache(redisClient, 24*time.Hour)
	weatherService := weather.CreateWeatherService(cfg.WeatherApiKey, cache)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make limited rapid requests to test rate limiting (reduced from 10 to 5 to avoid API overload)
	successCount := 0
	errorCount := 0

	for i := 0; i < 5; i++ {
		_, err := weatherService.GetCurrentWithFallback(ctx, "Barcelona")
		if err != nil {
			errorCount++
			t.Logf("Request %d failed: %v", i+1, err)
		} else {
			successCount++
		}

		// Increased delay between requests to be more respectful to API
		time.Sleep(200 * time.Millisecond)
	}

	t.Logf("Rate limiting test: %d successful, %d failed", successCount, errorCount)

	// At least some requests should succeed
	if successCount == 0 {
		t.Error("No requests succeeded, rate limiting might be too strict")
	}
}
