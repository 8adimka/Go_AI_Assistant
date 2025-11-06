package assistant

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/mongox"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PromptManager manages prompt configurations with caching and fallback
type PromptManager struct {
	cache    *redisx.Cache
	mongoDB  *mongo.Database
	fallback map[string]string
	cacheTTL time.Duration
}

// NewPromptManager creates a new prompt manager
func NewPromptManager(cfg *config.Config) *PromptManager {
	// Connect to MongoDB
	mongoDB := mongox.MustConnect(cfg.MongoURI, "tech_challenge")

	// Connect to Redis
	redisClient := redisx.MustConnect(cfg.RedisAddr)
	cacheTTL := time.Duration(cfg.CacheTTLHours) * time.Hour
	cache := redisx.NewCache(redisClient, cacheTTL)

	// Create fallback prompts from default configs
	fallback := make(map[string]string)
	defaultConfigs := model.GetDefaultPromptConfigs()
	for _, prompt := range defaultConfigs {
		fallback[prompt.Name] = prompt.Content
	}

	return &PromptManager{
		cache:    cache,
		mongoDB:  mongoDB,
		fallback: fallback,
		cacheTTL: cacheTTL,
	}
}

// GetPrompt retrieves a prompt by name with caching and fallback
func (pm *PromptManager) GetPrompt(ctx context.Context, name string) (string, error) {
	return pm.GetPromptWithPlatform(ctx, name, model.DefaultPlatform, model.DefaultUserSegment)
}

// GetPromptWithPlatform retrieves a prompt by name, platform, and user segment
func (pm *PromptManager) GetPromptWithPlatform(ctx context.Context, name, platform, userSegment string) (string, error) {
	// Generate cache key
	cacheKey := pm.generateCacheKey(name, platform, userSegment)

	// Try to get from Redis cache first
	var cachedPrompt string
	if err := pm.cache.Get(ctx, cacheKey, &cachedPrompt); err == nil {
		slog.DebugContext(ctx, "Prompt retrieved from cache",
			"name", name,
			"platform", platform,
			"user_segment", userSegment,
		)
		return cachedPrompt, nil
	} else if !errors.Is(err, redisx.ErrCacheMiss) {
		slog.WarnContext(ctx, "Cache error, proceeding without cache",
			"error", err,
			"name", name,
		)
	}

	// Try to get from MongoDB
	prompt, err := pm.getPromptFromMongo(ctx, name, platform, userSegment)
	if err == nil {
		// Cache the result
		if cacheErr := pm.cache.Set(ctx, cacheKey, prompt); cacheErr != nil {
			slog.WarnContext(ctx, "Failed to cache prompt",
				"error", cacheErr,
				"name", name,
			)
		}
		return prompt, nil
	}

	// If MongoDB fails, use fallback
	slog.WarnContext(ctx, "Failed to get prompt from MongoDB, using fallback",
		"name", name,
		"error", err,
	)

	if fallbackPrompt, exists := pm.fallback[name]; exists {
		return fallbackPrompt, nil
	}

	return "", fmt.Errorf("prompt not found: %s (no fallback available)", name)
}

// getPromptFromMongo retrieves a prompt from MongoDB
func (pm *PromptManager) getPromptFromMongo(ctx context.Context, name, platform, userSegment string) (string, error) {
	collection := pm.mongoDB.Collection("prompt_configs")

	// Build query to find active prompt with matching criteria
	filter := bson.M{
		"name":      name,
		"is_active": true,
		"$or": []bson.M{
			{"platform": platform},
			{"platform": model.DefaultPlatform},
		},
		"$and": []bson.M{
			{
				"$or": []bson.M{
					{"user_segment": userSegment},
					{"user_segment": model.DefaultUserSegment},
				},
			},
		},
	}

	// Sort by platform and user segment specificity (more specific first)
	sort := bson.D{
		{Key: "platform", Value: -1},     // Specific platform first
		{Key: "user_segment", Value: -1}, // Specific user segment first
		{Key: "updated_at", Value: -1},   // Most recent first
	}

	var promptConfig model.PromptConfig
	err := collection.FindOne(ctx, filter, options.FindOne().SetSort(sort)).Decode(&promptConfig)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", fmt.Errorf("no active prompt found for name: %s, platform: %s, user_segment: %s", name, platform, userSegment)
		}
		return "", fmt.Errorf("failed to query MongoDB for prompt: %w", err)
	}

	if promptConfig.Content == "" {
		return "", fmt.Errorf("prompt content is empty for name: %s", name)
	}

	slog.DebugContext(ctx, "Prompt retrieved from MongoDB",
		"name", name,
		"platform", platform,
		"user_segment", userSegment,
		"version", promptConfig.Version,
	)

	return promptConfig.Content, nil
}

// generateCacheKey generates a cache key for prompt
func (pm *PromptManager) generateCacheKey(name, platform, userSegment string) string {
	return fmt.Sprintf("prompt:%s:%s:%s", name, platform, userSegment)
}

// GetFallbackPrompt returns a fallback prompt by name
func (pm *PromptManager) GetFallbackPrompt(name string) (string, error) {
	if fallbackPrompt, exists := pm.fallback[name]; exists {
		return fallbackPrompt, nil
	}
	return "", fmt.Errorf("fallback prompt not found: %s", name)
}

// HealthCheck verifies that prompt system is working
func (pm *PromptManager) HealthCheck(ctx context.Context) error {
	// Test getting a system prompt
	_, err := pm.GetPrompt(ctx, model.PromptNameSystemPrompt)
	if err != nil {
		return fmt.Errorf("prompt system health check failed: %w", err)
	}

	// Test MongoDB connection
	collection := pm.mongoDB.Collection("prompt_configs")
	if err := collection.Database().Client().Ping(ctx, nil); err != nil {
		return fmt.Errorf("MongoDB connection failed: %w", err)
	}

	// Test Redis connection by setting and getting a test value
	testKey := "health_check:prompt_manager"
	testValue := "test"
	if err := pm.cache.Set(ctx, testKey, testValue); err != nil {
		return fmt.Errorf("Redis connection failed: %w", err)
	}

	var retrievedValue string
	if err := pm.cache.Get(ctx, testKey, &retrievedValue); err != nil {
		return fmt.Errorf("Redis read failed: %w", err)
	}

	if retrievedValue != testValue {
		return fmt.Errorf("Redis data integrity check failed")
	}

	// Clean up test key
	_ = pm.cache.Delete(ctx, testKey)

	return nil
}

// InitializePrompts ensures default prompts are available in MongoDB
func (pm *PromptManager) InitializePrompts(ctx context.Context) error {
	collection := pm.mongoDB.Collection("prompt_configs")

	defaultConfigs := model.GetDefaultPromptConfigs()

	for _, prompt := range defaultConfigs {
		// Check if prompt already exists
		filter := bson.M{
			"name":         prompt.Name,
			"platform":     prompt.Platform,
			"user_segment": prompt.UserSegment,
			"version":      prompt.Version,
		}

		var existingPrompt model.PromptConfig
		err := collection.FindOne(ctx, filter).Decode(&existingPrompt)

		if errors.Is(err, mongo.ErrNoDocuments) {
			// Insert new prompt
			_, err := collection.InsertOne(ctx, prompt)
			if err != nil {
				return fmt.Errorf("failed to insert prompt %s: %w", prompt.Name, err)
			}
			slog.InfoContext(ctx, "Inserted default prompt",
				"name", prompt.Name,
				"platform", prompt.Platform,
				"user_segment", prompt.UserSegment,
			)
		} else if err != nil {
			return fmt.Errorf("failed to check existing prompt %s: %w", prompt.Name, err)
		}
		// If prompt exists, do nothing
	}

	slog.InfoContext(ctx, "Prompt initialization completed")
	return nil
}
