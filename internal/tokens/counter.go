package tokens

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// TokenCounter provides accurate token counting using tiktoken
type TokenCounter struct {
	encoders map[string]*tiktoken.Tiktoken
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		encoders: make(map[string]*tiktoken.Tiktoken),
	}
}

// Count counts tokens for a given text and model
func (tc *TokenCounter) Count(ctx context.Context, text string, model string) (int, error) {
	encoder, err := tc.getEncoder(model)
	if err != nil {
		slog.WarnContext(ctx, "Failed to get encoder, using fallback estimation",
			"model", model, "error", err)
		return tc.fallbackEstimate(text), nil
	}

	tokens := encoder.Encode(text, nil, nil)
	return len(tokens), nil
}

// CountMessages counts tokens for a conversation
func (tc *TokenCounter) CountMessages(ctx context.Context, messages []Message, model string) (int, error) {
	totalTokens := 0

	for _, msg := range messages {
		// Count tokens for message content
		contentTokens, err := tc.Count(ctx, msg.Content, model)
		if err != nil {
			return 0, fmt.Errorf("failed to count tokens for message: %w", err)
		}

		// Add tokens for role and formatting (approximate)
		roleTokens := len(msg.Role) / 3 // Approximate tokens for role
		formattingTokens := 4           // Approximate tokens for message formatting

		totalTokens += contentTokens + roleTokens + formattingTokens
	}

	// Add tokens for system overhead
	totalTokens += 10

	return totalTokens, nil
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
	// This is better than the 4 chars per token used previously
	return len(text)/3 + 1
}

// Message represents a conversation message for token counting
type Message struct {
	Role    string
	Content string
}

// DefaultTokenCounter is a global instance for convenience
var DefaultTokenCounter = NewTokenCounter()

// CountText is a convenience function for counting tokens in text
func CountText(ctx context.Context, text string, model string) (int, error) {
	return DefaultTokenCounter.Count(ctx, text, model)
}

// CountMessages is a convenience function for counting tokens in messages
func CountMessages(ctx context.Context, messages []Message, model string) (int, error) {
	return DefaultTokenCounter.CountMessages(ctx, messages, model)
}
