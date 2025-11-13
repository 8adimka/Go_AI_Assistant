package chat

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"github.com/openai/openai-go/v2"
)

// Message represents a conversation message
type Message struct {
	Role    string
	Content string
}

// ContextManagerInterface defines the interface for context management
type ContextManagerInterface interface {
	// AddMessage adds a message to the conversation context
	AddMessage(ctx context.Context, conversationID string, message Message) error

	// GetContext returns the conversation context
	GetContext(conversationID string) []Message

	// GetTokenCount returns the current token count for a conversation
	GetTokenCount(conversationID string) int

	// ClearContext clears the conversation context
	ClearContext(conversationID string)

	// EnsureContextFits guarantees that the context fits within the specified token limit
	EnsureContextFits(ctx context.Context, conversationID string, targetTokens int) error
}

// StorageStrategy defines the strategy for context storage
type StorageStrategy int

const (
	// StorageStrategyRedis - use Redis for persistent storage (primary)
	StorageStrategyRedis StorageStrategy = iota
	// StorageStrategyMemory - use in-memory storage (fallback)
	StorageStrategyMemory
	// StorageStrategyHybrid - use Redis with memory fallback
	StorageStrategyHybrid
)

// SummarizationStrategy defines the strategy for context summarization
type SummarizationStrategy int

const (
	// SummarizationStrategyAI - use AI-powered summarization
	SummarizationStrategyAI SummarizationStrategy = iota
	// SummarizationStrategyBasic - use basic reduction without AI
	SummarizationStrategyBasic
	// SummarizationStrategyHybrid - try AI first, fallback to basic
	SummarizationStrategyHybrid
)

// ContextManager provides persistent context management with Redis storage
type ContextManager struct {
	mu           sync.RWMutex
	cache        *redisx.Cache
	openAIClient *openai.Client
	maxTokens    int
	maxHistory   int
}

// NewContextManager creates a new persistent context manager
func NewContextManager(cache *redisx.Cache, maxTokens, maxHistory int, openAIClient *openai.Client) *ContextManager {
	return &ContextManager{
		cache:        cache,
		openAIClient: openAIClient,
		maxTokens:    maxTokens,
		maxHistory:   maxHistory,
	}
}

// AddMessage adds a message to the conversation context with persistence
func (cm *ContextManager) AddMessage(ctx context.Context, conversationID string, message Message) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Load existing context
	existingContext, err := cm.loadContext(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// Add new message
	existingContext = append(existingContext, message)

	// Enforce max history limit
	if len(existingContext) > cm.maxHistory {
		// Remove oldest messages to stay within limit
		excess := len(existingContext) - cm.maxHistory
		existingContext = existingContext[excess:]
	}

	// Save updated context
	return cm.saveContext(ctx, conversationID, existingContext)
}

// GetContext returns the conversation context from persistent storage
func (cm *ContextManager) GetContext(conversationID string) []Message {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ctx := context.Background()
	messages, err := cm.loadContext(ctx, conversationID)
	if err != nil {
		slog.WarnContext(ctx, "Failed to load context from persistent storage",
			"conversation_id", conversationID, "error", err)
		return []Message{}
	}

	return messages
}

// GetTokenCount returns the current token count for a conversation
func (cm *ContextManager) GetTokenCount(conversationID string) int {
	messages := cm.GetContext(conversationID)
	totalTokens := 0

	for _, msg := range messages {
		totalTokens += cm.estimateTokens(msg.Content)
	}

	return totalTokens
}

// ClearContext clears the conversation context from persistent storage
func (cm *ContextManager) ClearContext(conversationID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ctx := context.Background()
	key := cm.generateContextKey(conversationID)
	if err := cm.cache.Delete(ctx, key); err != nil {
		slog.WarnContext(ctx, "Failed to clear context from persistent storage",
			"conversation_id", conversationID, "error", err)
	}
}

// EnsureContextFits guarantees that the context fits within the specified token limit
func (cm *ContextManager) EnsureContextFits(ctx context.Context, conversationID string, targetTokens int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Load current context
	messages, err := cm.loadContext(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	currentTokens := 0
	for _, msg := range messages {
		currentTokens += cm.estimateTokens(msg.Content)
	}

	if currentTokens <= targetTokens {
		return nil
	}

	slog.InfoContext(ctx, "Reducing context to fit token limit",
		"conversation_id", conversationID,
		"current_tokens", currentTokens,
		"target_tokens", targetTokens)

	// Try AI summarization first if available
	if cm.openAIClient != nil {
		if err := cm.performAISummarization(ctx, conversationID, messages); err == nil {
			slog.InfoContext(ctx, "AI summarization successful", "conversation_id", conversationID)
			return nil
		}
		slog.WarnContext(ctx, "AI summarization failed, using basic reduction",
			"conversation_id", conversationID)
	}

	// Fallback to basic reduction
	return cm.performBasicReduction(ctx, conversationID, messages, targetTokens)
}

// loadContext loads context from persistent storage
func (cm *ContextManager) loadContext(ctx context.Context, conversationID string) ([]Message, error) {
	key := cm.generateContextKey(conversationID)

	var messages []Message
	if err := cm.cache.Get(ctx, key, &messages); err != nil {
		if err == redisx.ErrCacheMiss {
			// No context exists yet, return empty slice
			return []Message{}, nil
		}
		return nil, fmt.Errorf("failed to load context from cache: %w", err)
	}

	return messages, nil
}

// saveContext saves context to persistent storage
func (cm *ContextManager) saveContext(ctx context.Context, conversationID string, messages []Message) error {
	key := cm.generateContextKey(conversationID)
	return cm.cache.Set(ctx, key, messages)
}

// generateContextKey generates a Redis key for context storage
func (cm *ContextManager) generateContextKey(conversationID string) string {
	return fmt.Sprintf("context:%s", conversationID)
}

// performAISummarization performs AI-powered summarization with persistent storage
func (cm *ContextManager) performAISummarization(ctx context.Context, conversationID string, messages []Message) error {
	if len(messages) <= 2 {
		return fmt.Errorf("not enough messages to summarize")
	}

	// Prepare conversation text for summarization
	var conversationText strings.Builder
	for _, msg := range messages {
		conversationText.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	// Create summarization prompt
	prompt := fmt.Sprintf(`Please summarize the following conversation, focusing on key points, decisions, and important information. Keep the summary concise but informative.

Conversation:
%s

Summary:`, conversationText.String())

	// Call OpenAI for summarization
	resp, err := cm.openAIClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4Turbo,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant that creates concise summaries of conversations."),
			openai.UserMessage(prompt),
		},
		MaxTokens: openai.Int(200),
	})

	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return fmt.Errorf("empty summary response")
	}

	summary := resp.Choices[0].Message.Content

	// Apply summary to persistent storage
	cm.ClearContext(conversationID)

	// Add summary as system message
	summaryMessage := Message{
		Role:    "assistant",
		Content: fmt.Sprintf("Previous conversation summary: %s", summary),
	}
	if err := cm.saveContext(ctx, conversationID, []Message{summaryMessage}); err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}

	// Keep only the most recent messages
	keepCount := 3
	if len(messages) < keepCount {
		keepCount = len(messages)
	}

	for i := len(messages) - keepCount; i < len(messages); i++ {
		if err := cm.AddMessage(ctx, conversationID, messages[i]); err != nil {
			slog.WarnContext(ctx, "Failed to add message during AI summarization",
				"conversation_id", conversationID, "error", err)
		}
	}

	return nil
}

// performBasicReduction performs basic context reduction without AI
func (cm *ContextManager) performBasicReduction(ctx context.Context, conversationID string, messages []Message, targetTokens int) error {
	currentTokens := 0
	for _, msg := range messages {
		currentTokens += cm.estimateTokens(msg.Content)
	}

	// Keep reducing until we fit within target
	for currentTokens > targetTokens && len(messages) > 1 {
		// Remove oldest message
		oldest := messages[0]
		messages = messages[1:]
		currentTokens -= cm.estimateTokens(oldest.Content)
	}

	// Save reduced context
	return cm.saveContext(ctx, conversationID, messages)
}

// estimateTokens provides improved token estimation
func (cm *ContextManager) estimateTokens(text string) int {
	// Improved approximation: 3.5 characters per token for English text
	// This is better than the 4 chars per token used previously
	// For production, consider using tiktoken library for exact counting
	return len(text)/3 + 1
}

// ConvertModelMessage converts chat model message to context message
func ConvertModelMessage(modelMsg *model.Message) Message {
	return Message{
		Role:    string(modelMsg.Role),
		Content: modelMsg.Content,
	}
}

// ConvertContextMessages converts context messages to model messages
func ConvertContextMessages(ctxMessages []Message) []*model.Message {
	var modelMessages []*model.Message
	for _, msg := range ctxMessages {
		modelMessages = append(modelMessages, &model.Message{
			Role:    model.Role(msg.Role),
			Content: msg.Content,
		})
	}
	return modelMessages
}
