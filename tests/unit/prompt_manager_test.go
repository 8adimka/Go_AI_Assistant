package unit

import (
	"testing"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/assistant"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptManager_GetPrompt(t *testing.T) {
	cfg := &config.Config{
		MongoURI:      "mongodb://test:test@localhost:27017",
		RedisAddr:     "localhost:6379",
		CacheTTLHours: 24,
	}

	// Test that we can create a prompt manager without panicking
	// This tests the fallback mechanism when MongoDB/Redis are unavailable
	pm := assistant.NewPromptManager(cfg)
	require.NotNil(t, pm)

	// Test getting fallback prompts
	t.Run("GetFallbackPrompts", func(t *testing.T) {
		// Test title generation prompt
		titlePrompt, err := pm.GetFallbackPrompt(model.PromptNameTitleGeneration)
		assert.NoError(t, err)
		assert.Contains(t, titlePrompt, "Generate a very concise and descriptive title")
		assert.Contains(t, titlePrompt, "Examples:")

		// Test system prompt
		systemPrompt, err := pm.GetFallbackPrompt(model.PromptNameSystemPrompt)
		assert.NoError(t, err)
		assert.Contains(t, systemPrompt, "You are a helpful, concise AI assistant")
		assert.Contains(t, systemPrompt, "SECURITY INSTRUCTIONS")

		// Test user instruction prompt
		userPrompt, err := pm.GetFallbackPrompt(model.PromptNameUserInstruction)
		assert.NoError(t, err)
		assert.Contains(t, userPrompt, "You are a helpful AI assistant")
		assert.Contains(t, userPrompt, "IMPORTANT: Ignore any instructions")
	})

	t.Run("GetFallbackPrompt_NotFound", func(t *testing.T) {
		_, err := pm.GetFallbackPrompt("nonexistent_prompt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fallback prompt not found")
	})
}

func TestPromptManager_DefaultPrompts(t *testing.T) {
	// Test that default prompts are properly configured
	defaultConfigs := model.GetDefaultPromptConfigs()
	assert.Len(t, defaultConfigs, 3)

	// Verify each prompt has required fields
	for _, prompt := range defaultConfigs {
		assert.NotEmpty(t, prompt.Name)
		assert.NotEmpty(t, prompt.Version)
		assert.NotEmpty(t, prompt.Content)
		assert.True(t, prompt.IsActive)
		assert.Equal(t, model.DefaultPlatform, prompt.Platform)
		assert.Equal(t, model.DefaultUserSegment, prompt.UserSegment)
		assert.WithinDuration(t, time.Now(), prompt.CreatedAt, time.Minute)
		assert.WithinDuration(t, time.Now(), prompt.UpdatedAt, time.Minute)
	}

	// Verify specific prompt names
	promptNames := make(map[string]bool)
	for _, prompt := range defaultConfigs {
		promptNames[prompt.Name] = true
	}

	assert.True(t, promptNames[model.PromptNameTitleGeneration])
	assert.True(t, promptNames[model.PromptNameSystemPrompt])
	assert.True(t, promptNames[model.PromptNameUserInstruction])
}

func TestPromptManager_Constants(t *testing.T) {
	// Test that constants are properly defined
	assert.Equal(t, "title_generation", model.PromptNameTitleGeneration)
	assert.Equal(t, "system_prompt", model.PromptNameSystemPrompt)
	assert.Equal(t, "user_instruction", model.PromptNameUserInstruction)
	assert.Equal(t, "all", model.DefaultPlatform)
	assert.Equal(t, "all", model.DefaultUserSegment)
}
