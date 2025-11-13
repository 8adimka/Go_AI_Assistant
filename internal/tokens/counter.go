package tokens

import (
	"fmt"
	"log/slog"
	"strings"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// TokenCounter provides accurate token counting using tiktoken
type TokenCounter struct {
	encoders map[string]*tiktoken.Tiktoken
	model    string
}

// NewTokenCounter creates a new token counter for a specific model
func NewTokenCounter(model string) (*TokenCounter, error) {
	tc := &TokenCounter{
		encoders: make(map[string]*tiktoken.Tiktoken),
		model:    model,
	}

	// Pre-initialize encoder for the specified model
	_, err := tc.getEncoder(model)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token counter for model %s: %w", model, err)
	}

	return tc, nil
}

// Count counts tokens for a given text
func (tc *TokenCounter) Count(text string) int {
	if text == "" {
		return 0
	}

	encoder, err := tc.getEncoder(tc.model)
	if err != nil {
		slog.Warn("Failed to get encoder, using fallback estimation",
			"model", tc.model, "error", err)
		return tc.fallbackEstimate(text)
	}

	tokens := encoder.Encode(text, nil, nil)
	return len(tokens)
}

// CountMessages counts tokens for a conversation
func (tc *TokenCounter) CountMessages(messages []Message) int {
	totalTokens := 0

	for _, msg := range messages {
		// Count tokens for message content
		contentTokens := tc.Count(msg.Content)

		// Add tokens for role and formatting (approximate)
		// In average ~4 tokens per message for role and formatting
		totalTokens += contentTokens + 4
	}

	return totalTokens
}

// CountPrompt counts tokens for system prompt
func (tc *TokenCounter) CountPrompt(prompt string) int {
	return tc.Count(prompt) + 2 // +2 tokens for system role
}

// EstimateContextSize estimates total context size
func (tc *TokenCounter) EstimateContextSize(systemPrompt string, messages []Message) int {
	return tc.CountPrompt(systemPrompt) + tc.CountMessages(messages)
}

// ValidateContextSize checks if context fits within limit
func (tc *TokenCounter) ValidateContextSize(systemPrompt string, messages []Message, maxTokens int) (bool, int) {
	totalTokens := tc.EstimateContextSize(systemPrompt, messages)
	return totalTokens <= maxTokens, totalTokens
}

// GetModel returns the model name
func (tc *TokenCounter) GetModel() string {
	return tc.model
}

// getEncoder gets or creates an encoder for the specified model
func (tc *TokenCounter) getEncoder(model string) (*tiktoken.Tiktoken, error) {
	// Map OpenAI model names to tiktoken encoding names
	encodingName := tc.getEncodingName(model)

	if encoder, exists := tc.encoders[encodingName]; exists {
		return encoder, nil
	}

	encoder, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding for model %s: %w", model, err)
	}

	tc.encoders[encodingName] = encoder
	return encoder, nil
}

// getEncodingName maps OpenAI model names to tiktoken encoding names
func (tc *TokenCounter) getEncodingName(model string) string {
	model = strings.ToLower(model)

	switch {
	case strings.Contains(model, "gpt-4"):
		return "cl100k_base"
	case strings.Contains(model, "gpt-3.5"):
		return "cl100k_base"
	case strings.Contains(model, "text-embedding"):
		return "cl100k_base"
	default:
		// Default to cl100k_base for newer models
		return "cl100k_base"
	}
}

// fallbackEstimate provides a fallback token estimation when tiktoken fails
func (tc *TokenCounter) fallbackEstimate(text string) int {
	// Improved approximation: 3.5 characters per token for English text
	return len(text)/3 + 1
}

// Message represents a conversation message for token counting
type Message struct {
	Role    string
	Content string
}

// GlobalTokenCounter is a global instance for default usage
var GlobalTokenCounter *TokenCounter

// InitGlobalTokenCounter initializes the global token counter
func InitGlobalTokenCounter(model string) error {
	counter, err := NewTokenCounter(model)
	if err != nil {
		return err
	}
	GlobalTokenCounter = counter
	slog.Info("Global token counter initialized", "model", model)
	return nil
}

// CountWithGlobal uses global counter for counting
func CountWithGlobal(text string) int {
	if GlobalTokenCounter == nil {
		// Fallback to simple heuristic if global not initialized
		return len(text)/3 + 1
	}
	return GlobalTokenCounter.Count(text)
}

// CountMessagesWithGlobal uses global counter for messages
func CountMessagesWithGlobal(messages []Message) int {
	if GlobalTokenCounter == nil {
		// Fallback to simple heuristic
		total := 0
		for _, msg := range messages {
			total += len(msg.Content)/3 + 1 + 4 // content + formatting
		}
		return total
	}
	return GlobalTokenCounter.CountMessages(messages)
}
