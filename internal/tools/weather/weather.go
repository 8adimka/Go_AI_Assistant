package weather

import (
	"context"
	"errors"
	"log/slog"

	"github.com/8adimka/Go_AI_Assistant/internal/tools/registry"
	"github.com/8adimka/Go_AI_Assistant/internal/weather"
)

// WeatherTool provides weather information using the weather service
type WeatherTool struct {
	weatherService *weather.FallbackWeatherService
}

// New creates a new WeatherTool instance
func New(weatherService *weather.FallbackWeatherService) *WeatherTool {
	return &WeatherTool{
		weatherService: weatherService,
	}
}

// Name returns the tool name
func (w *WeatherTool) Name() string {
	return "get_weather"
}

// Description returns the tool description
func (w *WeatherTool) Description() string {
	return "Get weather at the given location"
}

// Parameters returns the JSON schema for parameters
func (w *WeatherTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"location"},
	}
}

// Execute gets weather data for the specified location
func (w *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// Parse location from arguments
	location, ok := args["location"].(string)
	if !ok || location == "" {
		return "", errors.New("location is required")
	}

	slog.InfoContext(ctx, "Getting weather data", "location", location)

	// Get real weather data with fallback
	weatherData, err := w.weatherService.GetCurrentWithFallback(ctx, location)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get weather data", "location", location, "error", err)
		return "weather data unavailable", err
	}

	// Format weather data for response
	weatherMessage := weather.FormatWeather(weatherData)
	return weatherMessage, nil
}

// Ensure WeatherTool implements registry.Tool interface
var _ registry.Tool = (*WeatherTool)(nil)
