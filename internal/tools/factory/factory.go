package factory

import (
	"log/slog"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"github.com/8adimka/Go_AI_Assistant/internal/tools/datetime"
	"github.com/8adimka/Go_AI_Assistant/internal/tools/holidays"
	"github.com/8adimka/Go_AI_Assistant/internal/tools/registry"
	weathertool "github.com/8adimka/Go_AI_Assistant/internal/tools/weather"
	"github.com/8adimka/Go_AI_Assistant/internal/weather"
)

// Factory creates and registers all available tools
type Factory struct {
	registry *registry.ToolRegistry
	config   *config.Config
}

// NewFactory creates a new tool factory
func NewFactory(cfg *config.Config) *Factory {
	return &Factory{
		registry: registry.NewToolRegistry(),
		config:   cfg,
	}
}

// CreateAllTools creates and registers all available tools
func (f *Factory) CreateAllTools() *registry.ToolRegistry {
	slog.Info("Creating and registering tools")

	// Create Redis cache for weather service with configurable TTL
	redisClient := redisx.MustConnect(f.config.RedisAddr)
	cacheTTL := time.Duration(f.config.CacheTTLHours) * time.Hour
	cache := redisx.NewCache(redisClient, cacheTTL)

	// Create weather service with fallback
	weatherService := weather.CreateWeatherService(f.config.WeatherApiKey, cache)

	// Register all tools
	f.registerDateTimeTool()
	f.registerWeatherTool(weatherService)
	f.registerHolidaysTool()

	slog.Info("All tools registered successfully", "count", f.registry.Count())
	return f.registry
}

// registerDateTimeTool registers the date/time tool
func (f *Factory) registerDateTimeTool() {
	dateTimeTool := datetime.New()
	f.registry.Register(dateTimeTool)
}

// registerWeatherTool registers the weather tool
func (f *Factory) registerWeatherTool(weatherService *weather.FallbackWeatherService) {
	weatherTool := weathertool.New(weatherService)
	f.registry.Register(weatherTool)
}

// registerHolidaysTool registers the holidays tool
func (f *Factory) registerHolidaysTool() {
	// Use default calendar URL, can be overridden by environment variable
	calendarURL := "https://www.officeholidays.com/ics/spain/catalonia"
	holidaysTool := holidays.New(calendarURL)
	f.registry.Register(holidaysTool)
}

// GetRegistry returns the tool registry
func (f *Factory) GetRegistry() *registry.ToolRegistry {
	return f.registry
}
