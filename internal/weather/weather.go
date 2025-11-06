package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/retry"
	"golang.org/x/time/rate"
)

// WeatherData represents current weather information
type WeatherData struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature_c"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_kph"`
	WindDir     string  `json:"wind_dir"`
	Pressure    float64 `json:"pressure_mb"`
	FeelsLike   float64 `json:"feelslike_c"`
	Visibility  float64 `json:"vis_km"`
	UVIndex     float64 `json:"uv"`
	LastUpdated string  `json:"last_updated"`
}

// ForecastData represents weather forecast information
type ForecastData struct {
	Location string          `json:"location"`
	Forecast []ForecastDay   `json:"forecast"`
}

// ForecastDay represents daily forecast
type ForecastDay struct {
	Date        string  `json:"date"`
	MaxTemp     float64 `json:"maxtemp_c"`
	MinTemp     float64 `json:"mintemp_c"`
	AvgTemp     float64 `json:"avgtemp_c"`
	Condition   string  `json:"condition"`
	MaxWind     float64 `json:"maxwind_kph"`
	TotalPrecip float64 `json:"totalprecip_mm"`
	AvgHumidity int     `json:"avghumidity"`
	UVIndex     float64 `json:"uv"`
}

// WeatherProvider interface defines weather data operations
type WeatherProvider interface {
	GetCurrent(ctx context.Context, location string) (*WeatherData, error)
	GetForecast(ctx context.Context, location string, days int) (*ForecastData, error)
}

// WeatherAPIClient implements WeatherProvider using WeatherAPI.com
type WeatherAPIClient struct {
	client      *http.Client
	apiKey      string
	baseURL     string
	rateLimiter *rate.Limiter
	retryConfig retry.RetryConfig
}

// NewWeatherAPIClient creates a new WeatherAPI client with rate limiting
func NewWeatherAPIClient(apiKey string) *WeatherAPIClient {
	cfg := config.Load()
	return &WeatherAPIClient{
		client:      &http.Client{Timeout: 10 * time.Second},
		apiKey:      apiKey,
		baseURL:     "http://api.weatherapi.com/v1",
		rateLimiter: rate.NewLimiter(rate.Every(time.Minute), 10), // 10 requests per minute
		retryConfig: retry.ConfigFromAppConfig(cfg),
	}
}

// GetCurrent retrieves current weather for a location
func (w *WeatherAPIClient) GetCurrent(ctx context.Context, location string) (*WeatherData, error) {
	// Apply rate limiting
	if err := w.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/current.json?key=%s&q=%s&aqi=no", w.baseURL, w.apiKey, location)
	
	// Use retry logic for HTTP request
	resp, err := retry.RetryWithResult(ctx, w.retryConfig, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		
		resp, err := w.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		
		// Check for retryable status codes
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			return nil, fmt.Errorf("retryable HTTP error: %s", resp.Status)
		}
		
		return resp, nil
	})
	
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			return nil, fmt.Errorf("invalid location: %s", location)
		}
		return nil, fmt.Errorf("weather API error: %s", resp.Status)
	}

	var apiResponse struct {
		Location struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"location"`
		Current struct {
			TempC      float64 `json:"temp_c"`
			Condition  struct {
				Text string `json:"text"`
			} `json:"condition"`
			Humidity    int     `json:"humidity"`
			WindKph     float64 `json:"wind_kph"`
			WindDir     string  `json:"wind_dir"`
			PressureMb  float64 `json:"pressure_mb"`
			FeelslikeC  float64 `json:"feelslike_c"`
			VisKm       float64 `json:"vis_km"`
			UV          float64 `json:"uv"`
			LastUpdated string  `json:"last_updated"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	weather := &WeatherData{
		Location:    fmt.Sprintf("%s, %s", apiResponse.Location.Name, apiResponse.Location.Country),
		Temperature: apiResponse.Current.TempC,
		Condition:   apiResponse.Current.Condition.Text,
		Humidity:    apiResponse.Current.Humidity,
		WindSpeed:   apiResponse.Current.WindKph,
		WindDir:     apiResponse.Current.WindDir,
		Pressure:    apiResponse.Current.PressureMb,
		FeelsLike:   apiResponse.Current.FeelslikeC,
		Visibility:  apiResponse.Current.VisKm,
		UVIndex:     apiResponse.Current.UV,
		LastUpdated: apiResponse.Current.LastUpdated,
	}

	slog.InfoContext(ctx, "Retrieved current weather", "location", location, "temperature", weather.Temperature)
	return weather, nil
}

// GetForecast retrieves weather forecast for a location
func (w *WeatherAPIClient) GetForecast(ctx context.Context, location string, days int) (*ForecastData, error) {
	if days < 1 || days > 10 {
		return nil, fmt.Errorf("days must be between 1 and 10")
	}

	// Apply rate limiting
	if err := w.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/forecast.json?key=%s&q=%s&days=%d&aqi=no", w.baseURL, w.apiKey, location, days)
	
	// Use retry logic for HTTP request
	resp, err := retry.RetryWithResult(ctx, w.retryConfig, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		
		resp, err := w.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		
		// Check for retryable status codes
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			return nil, fmt.Errorf("retryable HTTP error: %s", resp.Status)
		}
		
		return resp, nil
	})
	
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			return nil, fmt.Errorf("invalid location: %s", location)
		}
		return nil, fmt.Errorf("weather API error: %s", resp.Status)
	}

	var apiResponse struct {
		Location struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"location"`
		Forecast struct {
			Forecastday []struct {
				Date string `json:"date"`
				Day  struct {
					MaxtempC      float64 `json:"maxtemp_c"`
					MintempC      float64 `json:"mintemp_c"`
					AvgtempC      float64 `json:"avgtemp_c"`
					Condition     struct {
						Text string `json:"text"`
					} `json:"condition"`
					MaxwindKph    float64 `json:"maxwind_kph"`
					TotalprecipMm float64 `json:"totalprecip_mm"`
					Avghumidity   int     `json:"avghumidity"`
					UV            float64 `json:"uv"`
				} `json:"day"`
			} `json:"forecastday"`
		} `json:"forecast"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	forecast := &ForecastData{
		Location: fmt.Sprintf("%s, %s", apiResponse.Location.Name, apiResponse.Location.Country),
		Forecast: make([]ForecastDay, 0, len(apiResponse.Forecast.Forecastday)),
	}

	for _, day := range apiResponse.Forecast.Forecastday {
		forecast.Forecast = append(forecast.Forecast, ForecastDay{
			Date:        day.Date,
			MaxTemp:     day.Day.MaxtempC,
			MinTemp:     day.Day.MintempC,
			AvgTemp:     day.Day.AvgtempC,
			Condition:   day.Day.Condition.Text,
			MaxWind:     day.Day.MaxwindKph,
			TotalPrecip: day.Day.TotalprecipMm,
			AvgHumidity: day.Day.Avghumidity,
			UVIndex:     day.Day.UV,
		})
	}

	slog.InfoContext(ctx, "Retrieved weather forecast", "location", location, "days", days)
	return forecast, nil
}

// FormatWeather formats weather data for display
func FormatWeather(weather *WeatherData) string {
	if weather == nil {
		return "Weather data unavailable"
	}

	return fmt.Sprintf(
		"Current weather in %s: %s, %.1f°C (feels like %.1f°C). "+
			"Humidity: %d%%, Wind: %.1f km/h %s, Pressure: %.0f mb, "+
			"Visibility: %.1f km, UV Index: %.1f. Last updated: %s",
		weather.Location,
		strings.ToLower(weather.Condition),
		weather.Temperature,
		weather.FeelsLike,
		weather.Humidity,
		weather.WindSpeed,
		weather.WindDir,
		weather.Pressure,
		weather.Visibility,
		weather.UVIndex,
		weather.LastUpdated,
	)
}

// FormatForecast formats forecast data for display
func FormatForecast(forecast *ForecastData, days int) string {
	if forecast == nil || len(forecast.Forecast) == 0 {
		return "Forecast data unavailable"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Weather forecast for %s:\n", forecast.Location))

	for i, day := range forecast.Forecast {
		if i >= days {
			break
		}
		builder.WriteString(fmt.Sprintf(
			"- %s: %s, %.1f°C to %.1f°C (avg %.1f°C), Wind: %.1f km/h, "+
				"Precip: %.1f mm, Humidity: %d%%, UV: %.1f\n",
			day.Date,
			strings.ToLower(day.Condition),
			day.MinTemp,
			day.MaxTemp,
			day.AvgTemp,
			day.MaxWind,
			day.TotalPrecip,
			day.AvgHumidity,
			day.UVIndex,
		))
	}

	return builder.String()
}
