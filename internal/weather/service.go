package weather

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
)

// WeatherService provides weather data with caching
type WeatherService struct {
	provider WeatherProvider
	cache    *redisx.Cache
}

// NewWeatherService creates a new weather service with caching
func NewWeatherService(provider WeatherProvider, cache *redisx.Cache) *WeatherService {
	return &WeatherService{
		provider: provider,
		cache:    cache,
	}
}

// GetCurrentWithCache retrieves current weather with Redis caching
func (s *WeatherService) GetCurrentWithCache(ctx context.Context, location string) (*WeatherData, error) {
	// Generate cache key
	cacheKey := s.cache.GenerateKey("weather:current", location)
	
	// Try to get from cache first
	var cachedWeather WeatherData
	if err := s.cache.Get(ctx, cacheKey, &cachedWeather); err == nil {
		slog.InfoContext(ctx, "Weather data retrieved from cache", "location", location)
		return &cachedWeather, nil
	} else if !errors.Is(err, redisx.ErrCacheMiss) {
		slog.WarnContext(ctx, "Cache error, proceeding without cache", "error", err)
	}

	// Get fresh data from provider
	weather, err := s.provider.GetCurrent(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather data: %w", err)
	}

	// Cache the result for 1 hour (weather doesn't change frequently)
	if err := s.cache.Set(ctx, cacheKey, weather); err != nil {
		slog.WarnContext(ctx, "Failed to cache weather data", "error", err)
	}

	slog.InfoContext(ctx, "Weather data retrieved from API and cached", "location", location)
	return weather, nil
}

// GetForecastWithCache retrieves weather forecast with Redis caching
func (s *WeatherService) GetForecastWithCache(ctx context.Context, location string, days int) (*ForecastData, error) {
	// Generate cache key
	cacheKey := s.cache.GenerateKey("weather:forecast", fmt.Sprintf("%s:%d", location, days))
	
	// Try to get from cache first
	var cachedForecast ForecastData
	if err := s.cache.Get(ctx, cacheKey, &cachedForecast); err == nil {
		slog.InfoContext(ctx, "Forecast data retrieved from cache", "location", location, "days", days)
		return &cachedForecast, nil
	} else if !errors.Is(err, redisx.ErrCacheMiss) {
		slog.WarnContext(ctx, "Cache error, proceeding without cache", "error", err)
	}

	// Get fresh data from provider
	forecast, err := s.provider.GetForecast(ctx, location, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get forecast data: %w", err)
	}

	// Cache the result for 3 hours (forecast changes less frequently)
	if err := s.cache.Set(ctx, cacheKey, forecast); err != nil {
		slog.WarnContext(ctx, "Failed to cache forecast data", "error", err)
	}

	slog.InfoContext(ctx, "Forecast data retrieved from API and cached", "location", location, "days", days)
	return forecast, nil
}

// MockWeatherProvider provides mock weather data for testing and fallback
type MockWeatherProvider struct{}

// NewMockWeatherProvider creates a new mock weather provider
func NewMockWeatherProvider() *MockWeatherProvider {
	return &MockWeatherProvider{}
}

// GetCurrent returns mock current weather data
func (m *MockWeatherProvider) GetCurrent(ctx context.Context, location string) (*WeatherData, error) {
	slog.WarnContext(ctx, "Using mock weather data", "location", location)
	
	return &WeatherData{
		Location:    location,
		Temperature: 20.0,
		Condition:   "Partly cloudy",
		Humidity:    65,
		WindSpeed:   15.0,
		WindDir:     "NW",
		Pressure:    1013.0,
		FeelsLike:   19.5,
		Visibility:  10.0,
		UVIndex:     5.0,
		LastUpdated: time.Now().Format(time.RFC3339),
	}, nil
}

// GetForecast returns mock forecast data
func (m *MockWeatherProvider) GetForecast(ctx context.Context, location string, days int) (*ForecastData, error) {
	slog.WarnContext(ctx, "Using mock forecast data", "location", location, "days", days)
	
	if days < 1 || days > 10 {
		return nil, fmt.Errorf("days must be between 1 and 10")
	}

	forecast := &ForecastData{
		Location: location,
		Forecast: make([]ForecastDay, days),
	}

	now := time.Now()
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, i).Format("2006-01-02")
		forecast.Forecast[i] = ForecastDay{
			Date:        date,
			MaxTemp:     22.0 + float64(i),
			MinTemp:     15.0 - float64(i),
			AvgTemp:     18.5,
			Condition:   "Partly cloudy",
			MaxWind:     20.0,
			TotalPrecip: 0.0,
			AvgHumidity: 70,
			UVIndex:     5.0,
		}
	}

	return forecast, nil
}

// FallbackWeatherService provides weather data with fallback to mock data
type FallbackWeatherService struct {
	primaryProvider WeatherProvider
	fallbackProvider WeatherProvider
	cache           *redisx.Cache
}

// NewFallbackWeatherService creates a weather service with fallback
func NewFallbackWeatherService(primary WeatherProvider, fallback WeatherProvider, cache *redisx.Cache) *FallbackWeatherService {
	return &FallbackWeatherService{
		primaryProvider: primary,
		fallbackProvider: fallback,
		cache:           cache,
	}
}

// GetCurrentWithFallback tries primary provider, falls back to mock data on error
func (f *FallbackWeatherService) GetCurrentWithFallback(ctx context.Context, location string) (*WeatherData, error) {
	// Try primary provider first
	weather, err := f.primaryProvider.GetCurrent(ctx, location)
	if err == nil {
		return weather, nil
	}

	slog.ErrorContext(ctx, "Primary weather provider failed, using fallback", 
		"location", location, "error", err)

	// Fall back to mock provider
	return f.fallbackProvider.GetCurrent(ctx, location)
}

// GetForecastWithFallback tries primary provider, falls back to mock data on error
func (f *FallbackWeatherService) GetForecastWithFallback(ctx context.Context, location string, days int) (*ForecastData, error) {
	// Try primary provider first
	forecast, err := f.primaryProvider.GetForecast(ctx, location, days)
	if err == nil {
		return forecast, nil
	}

	slog.ErrorContext(ctx, "Primary forecast provider failed, using fallback", 
		"location", location, "days", days, "error", err)

	// Fall back to mock provider
	return f.fallbackProvider.GetForecast(ctx, location, days)
}

// Helper function to create weather service with all features
func CreateWeatherService(apiKey string, cache *redisx.Cache) *FallbackWeatherService {
	var primaryProvider WeatherProvider
	
	if apiKey != "" {
		primaryProvider = NewWeatherAPIClient(apiKey)
	} else {
		slog.Warn("No WeatherAPI key provided, using mock provider as primary")
		primaryProvider = NewMockWeatherProvider()
	}

	fallbackProvider := NewMockWeatherProvider()
	
	return NewFallbackWeatherService(primaryProvider, fallbackProvider, cache)
}
