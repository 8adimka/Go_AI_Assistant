package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/metrics"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"github.com/8adimka/Go_AI_Assistant/internal/retry"
	"github.com/8adimka/Go_AI_Assistant/internal/tokens"
	"github.com/8adimka/Go_AI_Assistant/internal/tools/factory"
	"github.com/8adimka/Go_AI_Assistant/internal/tools/registry"
	"github.com/openai/openai-go"
)

// UnifiedAssistant provides comprehensive context management with AI summarization
type UnifiedAssistant struct {
	cli            openai.Client
	cache          *redisx.Cache
	toolRegistry   *registry.ToolRegistry
	retryConfig    retry.RetryConfig
	metrics        *metrics.Metrics
	promptManager  *PromptManager
	contextManager chat.ContextManagerInterface
	cfg            *config.Config
	fallbackMode   bool // Graceful degradation mode
}

// New creates a new unified assistant with enhanced context management
func New(appMetrics *metrics.Metrics) *UnifiedAssistant {
	// Load configuration
	cfg := config.Load()
	redisClient := redisx.MustConnect(cfg.RedisAddr)

	// Use configurable cache TTL from config
	cacheTTL := time.Duration(cfg.CacheTTLHours) * time.Hour
	cache := redisx.NewCache(redisClient, cacheTTL)

	// Create tool registry with all available tools
	toolFactory := factory.NewFactory(cfg)
	toolRegistry := toolFactory.CreateAllTools()

	// Create prompt manager
	promptManager := NewPromptManager(cfg)

	// Create context manager with configurable limits
	maxTokens := 4000
	if cfg.MaxContextTokens > 0 {
		maxTokens = cfg.MaxContextTokens
	}
	maxHistory := 50 // Maximum number of messages to keep

	// Create Redis cache for context management with configurable TTL
	contextTTL := time.Duration(cfg.CacheTTLHours) * time.Hour
	contextCache := redisx.NewCache(redisClient, contextTTL)

	// Use the actual OpenAI client for summarization
	openAIClient := openai.NewClient()

	// Create token counter for precise token counting
	tokenCounter, err := tokens.NewTokenCounter(cfg.OpenAIModel)
	if err != nil {
		slog.Warn("Failed to create precise token counter, using fallback", "error", err)
		tokenCounter = nil
	}

	// Use context manager with Redis storage and token counter
	contextManager := chat.NewContextManager(
		contextCache,
		maxTokens,
		maxHistory,
		tokenCounter,
	)

	return &UnifiedAssistant{
		cli:            openAIClient,
		cache:          cache,
		toolRegistry:   toolRegistry,
		retryConfig:    retry.ConfigFromAppConfig(cfg),
		metrics:        appMetrics,
		promptManager:  promptManager,
		contextManager: contextManager,
		cfg:            cfg,
	}
}

// Title generates a conversation title with enhanced logging
func (ua *UnifiedAssistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "An empty conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation",
		"conversation_id", conv.ID.Hex(),
		"user_id", conv.UserID,
		"platform", conv.Platform,
	)

	// Try to get from cache first
	userMessage := conv.Messages[0].Content
	cacheKey := ua.cache.GenerateKey("title", userMessage)

	var cachedTitle string
	if err := ua.cache.Get(ctx, cacheKey, &cachedTitle); err == nil {
		slog.InfoContext(ctx, "Title retrieved from cache",
			"conversation_id", conv.ID.Hex(),
			"user_id", conv.UserID,
		)
		return cachedTitle, nil
	} else if !errors.Is(err, redisx.ErrCacheMiss) {
		slog.WarnContext(ctx, "Cache error, proceeding without cache", "error", err)
	}

	// Get title generation prompt from prompt manager
	titlePrompt, err := ua.promptManager.GetPromptWithPlatform(ctx, model.PromptNameTitleGeneration, conv.Platform, conv.UserID)
	if err != nil {
		slog.WarnContext(ctx, "Failed to get title prompt, using fallback", "error", err)
		// Use fallback prompt from manager
		titlePrompt, err = ua.promptManager.GetFallbackPrompt(model.PromptNameTitleGeneration)
		if err != nil {
			return "", fmt.Errorf("failed to get fallback title prompt: %w", err)
		}
	}

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(titlePrompt),
		openai.UserMessage(userMessage),
	}

	// Use retry logic for OpenAI API call with timing
	start := time.Now()
	resp, err := retry.RetryWithResult(ctx, ua.retryConfig, func() (*openai.ChatCompletion, error) {
		return ua.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:     openai.ChatModelGPT4Turbo, // Faster model for titles
			Messages:  msgs,
			MaxTokens: openai.Int(30), // Limit tokens for brevity
		})
	})
	duration := time.Since(start)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	// Record OpenAI metrics with token usage
	if ua.metrics != nil {
		ua.metrics.RecordOpenAIRequestWithTokens(ctx, "title", string(openai.ChatModelGPT4Turbo),
			conv.UserID, conv.Platform, duration,
			int64(resp.Usage.PromptTokens), int64(resp.Usage.CompletionTokens), int64(resp.Usage.TotalTokens))
	}

	// Log OpenAI API call with token usage
	slog.InfoContext(ctx, "OpenAI API call completed",
		"operation", "title",
		"model", openai.ChatModelGPT4Turbo,
		"conversation_id", conv.ID.Hex(),
		"user_id", conv.UserID,
		"platform", conv.Platform,
		"prompt_tokens", resp.Usage.PromptTokens,
		"completion_tokens", resp.Usage.CompletionTokens,
		"total_tokens", resp.Usage.TotalTokens,
		"duration_ms", duration.Milliseconds(),
	)

	title := resp.Choices[0].Message.Content
	title = ua.formatTitle(title)

	// Save to cache
	if err := ua.cache.Set(ctx, cacheKey, title); err != nil {
		slog.WarnContext(ctx, "Failed to cache title", "error", err)
	}

	return title, nil
}

// Reply generates a reply with intelligent context management and AI summarization
func (ua *UnifiedAssistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation",
		"conversation_id", conv.ID.Hex(),
		"user_id", conv.UserID,
		"platform", conv.Platform,
		"messages_count", len(conv.Messages),
	)

	// Get system prompt from prompt manager
	systemPrompt, err := ua.promptManager.GetPromptWithPlatform(ctx, model.PromptNameSystemPrompt, conv.Platform, conv.UserID)
	if err != nil {
		slog.WarnContext(ctx, "Failed to get system prompt, using fallback", "error", err)
		// Use fallback prompt from manager
		systemPrompt, err = ua.promptManager.GetFallbackPrompt(model.PromptNameSystemPrompt)
		if err != nil {
			return "", fmt.Errorf("failed to get fallback system prompt: %w", err)
		}
	}

	// Use context manager to manage conversation context with token limits
	conversationID := conv.ID.Hex()

	// Add all existing messages to context manager
	for _, msg := range conv.Messages {
		contextMsg := chat.ConvertModelMessage(msg)
		if err := ua.contextManager.AddMessage(ctx, conversationID, contextMsg); err != nil {
			slog.WarnContext(ctx, "Failed to add message to context manager",
				"conversation_id", conversationID, "error", err)
		}
	}

	// Get managed context from context manager
	managedContext := ua.contextManager.GetContext(conversationID)
	currentTokenCount := ua.contextManager.GetTokenCount(conversationID)

	slog.InfoContext(ctx, "Context manager state",
		"conversation_id", conversationID,
		"managed_messages", len(managedContext),
		"current_tokens", currentTokenCount,
	)

	// Build messages for OpenAI API using managed context
	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	for _, msg := range managedContext {
		switch msg.Role {
		case "user":
			msgs = append(msgs, openai.UserMessage(msg.Content))
		case "assistant":
			msgs = append(msgs, openai.AssistantMessage(msg.Content))
		}
	}

	// Convert registered tools to OpenAI tool format
	tools := ua.convertToolsToOpenAIFormat()

	// Calculate estimated token count for the current context
	estimatedTokens := ua.estimateTokenCount(msgs, tools)

	// Check if context exceeds safe limits for the model
	maxModelTokens := ua.getMaxTokensForModel(openai.ChatModelGPT4_1)
	if estimatedTokens > maxModelTokens {
		slog.WarnContext(ctx, "Context exceeds model limits, performing proactive reduction",
			"conversation_id", conversationID,
			"estimated_tokens", estimatedTokens,
			"model_max_tokens", maxModelTokens,
			"model", openai.ChatModelGPT4_1)

		// Use context manager to ensure context fits within model limits
		// Use 90% of model limit to be safe
		safeLimit := int(float64(maxModelTokens) * 0.9)
		if err := ua.contextManager.EnsureContextFits(ctx, conversationID, safeLimit); err != nil {
			return "", fmt.Errorf("failed to reduce context size: %w", err)
		}

		// Rebuild messages with reduced context
		managedContext = ua.contextManager.GetContext(conversationID)
		msgs = []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
		}
		for _, msg := range managedContext {
			switch msg.Role {
			case "user":
				msgs = append(msgs, openai.UserMessage(msg.Content))
			case "assistant":
				msgs = append(msgs, openai.AssistantMessage(msg.Content))
			}
		}

		// Recalculate token count
		estimatedTokens = ua.estimateTokenCount(msgs, tools)
		slog.InfoContext(ctx, "Context reduced after proactive reduction",
			"conversation_id", conversationID,
			"new_estimated_tokens", estimatedTokens,
			"model_max_tokens", maxModelTokens)
	}

	// Enhanced retry mechanism with intelligent context reduction
	// Reduced from 15 to 5 iterations for better performance
	for i := 0; i < 5; i++ {
		// Use retry logic for OpenAI API call with timing
		start := time.Now()
		resp, err := retry.RetryWithResult(ctx, ua.retryConfig, func() (*openai.ChatCompletion, error) {
			return ua.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
				Model:    openai.ChatModelGPT4_1,
				Messages: msgs,
				Tools:    tools,
			})
		})
		duration := time.Since(start)

		if err != nil {
			// Check if error is due to context length exceeded
			if ua.isContextLengthExceededError(err) {
				slog.WarnContext(ctx, "Context length exceeded, performing emergency reduction",
					"conversation_id", conversationID,
					"iteration", i+1)

				// Use context manager to guarantee context fits within model limits
				// Use 80% of model limit to be extra safe after error
				safeLimit := int(float64(maxModelTokens) * 0.8)
				if err := ua.contextManager.EnsureContextFits(ctx, conversationID, safeLimit); err != nil {
					return "", fmt.Errorf("failed to reduce context after length exceeded: %w", err)
				}

				// Rebuild messages with reduced context
				managedContext = ua.contextManager.GetContext(conversationID)
				msgs = []openai.ChatCompletionMessageParamUnion{
					openai.SystemMessage(systemPrompt),
				}
				for _, msg := range managedContext {
					switch msg.Role {
					case "user":
						msgs = append(msgs, openai.UserMessage(msg.Content))
					case "assistant":
						msgs = append(msgs, openai.AssistantMessage(msg.Content))
					}
				}

				// Recalculate token count
				estimatedTokens = ua.estimateTokenCount(msgs, tools)
				slog.InfoContext(ctx, "Context reduced after length exceeded error",
					"conversation_id", conversationID,
					"new_estimated_tokens", estimatedTokens,
					"safe_limit", safeLimit)

				// Continue to next iteration to retry
				continue
			}
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		// Record OpenAI metrics with token usage
		if ua.metrics != nil {
			ua.metrics.RecordOpenAIRequestWithTokens(ctx, "reply", string(openai.ChatModelGPT4_1),
				conv.UserID, conv.Platform, duration,
				int64(resp.Usage.PromptTokens), int64(resp.Usage.CompletionTokens), int64(resp.Usage.TotalTokens))

			// Record context token count
			ua.metrics.RecordContextTokenCount(ctx, conversationID, conv.Platform, int64(currentTokenCount))

			// Record token estimation error
			ua.metrics.RecordTokenEstimationError(ctx, "reply", estimatedTokens, int(resp.Usage.PromptTokens))
		}

		// Log OpenAI API call with token usage
		slog.InfoContext(ctx, "OpenAI API call completed",
			"operation", "reply",
			"model", openai.ChatModelGPT4_1,
			"conversation_id", conv.ID.Hex(),
			"user_id", conv.UserID,
			"platform", conv.Platform,
			"iteration", i+1,
			"prompt_tokens", resp.Usage.PromptTokens,
			"completion_tokens", resp.Usage.CompletionTokens,
			"total_tokens", resp.Usage.TotalTokens,
			"duration_ms", duration.Milliseconds(),
			"has_tool_calls", len(resp.Choices[0].Message.ToolCalls) > 0,
			"context_tokens", currentTokenCount,
		)

		if message := resp.Choices[0].Message; len(message.ToolCalls) > 0 {
			msgs = append(msgs, message.ToParam())

			for _, call := range message.ToolCalls {
				slog.InfoContext(ctx, "Tool call received",
					"conversation_id", conv.ID.Hex(),
					"tool_name", call.Function.Name,
					"args", call.Function.Arguments,
				)

				// Execute tool using the registry
				result, err := ua.executeTool(ctx, call.Function.Name, call.Function.Arguments)
				if err != nil {
					slog.ErrorContext(ctx, "Tool execution failed",
						"conversation_id", conv.ID.Hex(),
						"tool_name", call.Function.Name,
						"error", err,
					)
					msgs = append(msgs, openai.ToolMessage("tool execution failed: "+err.Error(), call.ID))
				} else {
					msgs = append(msgs, openai.ToolMessage(result, call.ID))
				}
			}

			continue
		}

		// Add assistant's response to context manager
		assistantMsg := chat.ConvertModelMessage(&model.Message{
			Role:    model.RoleAssistant,
			Content: resp.Choices[0].Message.Content,
		})
		if err := ua.contextManager.AddMessage(ctx, conversationID, assistantMsg); err != nil {
			slog.WarnContext(ctx, "Failed to add assistant message to context manager",
				"conversation_id", conversationID, "error", err)
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}

// formatTitle formats and validates the title
func (ua *UnifiedAssistant) formatTitle(title string) string {
	// Remove extra spaces and newlines
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "\n", " ")

	// Remove quotes and other special characters
	title = strings.Trim(title, " \"'`-")

	// Limit length
	if len(title) > 60 {
		title = title[:60]
	}

	// Convert to Title Case
	title = ua.toTitleCase(title)

	return title
}

// toTitleCase converts string to Title Case
func (ua *UnifiedAssistant) toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			// All words except short conjunctions and prepositions get capitalized
			shortWords := map[string]bool{
				"a": true, "an": true, "the": true, "and": true, "but": true, "or": true,
				"for": true, "nor": true, "on": true, "at": true, "to": true, "by": true,
				"in": true, "of": true, "with": true,
			}

			// First word is always capitalized
			if i == 0 || !shortWords[strings.ToLower(word)] {
				words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			} else {
				words[i] = strings.ToLower(word)
			}
		}
	}
	return strings.Join(words, " ")
}

// convertToolsToOpenAIFormat converts registered tools to OpenAI tool format
func (ua *UnifiedAssistant) convertToolsToOpenAIFormat() []openai.ChatCompletionToolParam {
	var tools []openai.ChatCompletionToolParam

	for _, tool := range ua.toolRegistry.GetAll() {
		tools = append(tools, openai.ChatCompletionToolParam{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name(),
				Description: openai.String(tool.Description()),
				Parameters:  openai.FunctionParameters(tool.Parameters()),
			},
		})
	}

	return tools
}

// executeTool executes a tool by name with the provided arguments
func (ua *UnifiedAssistant) executeTool(ctx context.Context, toolName string, arguments string) (string, error) {
	tool := ua.toolRegistry.Get(toolName)
	if tool == nil {
		return "", errors.New("unknown tool: " + toolName)
	}

	// Parse JSON arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", errors.New("failed to parse tool arguments: " + err.Error())
	}

	// Execute the tool
	return tool.Execute(ctx, args)
}

// estimateTokenCount estimates the total token count for messages and tools
func (ua *UnifiedAssistant) estimateTokenCount(msgs []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) int {
	totalTokens := 0

	// Simple but improved approximation: convert all messages to JSON string and count characters
	// This is more reliable than complex type switching
	msgStr := fmt.Sprintf("%v", msgs)
	totalTokens += len(msgStr) / 3 // Improved: 3 chars per token for better accuracy

	// Estimate tokens for tools
	toolStr := fmt.Sprintf("%v", tools)
	totalTokens += len(toolStr) / 3

	// Add buffer for system overhead and formatting
	totalTokens += 150

	return totalTokens
}

// getMaxTokensForModel returns the maximum context tokens for a given model
func (ua *UnifiedAssistant) getMaxTokensForModel(model openai.ChatModel) int {
	// Model-specific token limits (conservative estimates)
	modelLimits := map[openai.ChatModel]int{
		openai.ChatModelGPT4_1:      128000, // GPT-4-128K
		openai.ChatModelGPT4Turbo:   128000, // GPT-4 Turbo
		openai.ChatModelGPT4:        8192,   // GPT-4
		openai.ChatModelGPT4_0613:   8192,   // GPT-4 (June 2023)
		openai.ChatModelGPT4_0314:   8192,   // GPT-4 (March 2023)
		openai.ChatModelGPT4_32k:    32768,  // GPT-4 32K
		openai.ChatModelGPT3_5Turbo: 4096,   // GPT-3.5 Turbo
	}

	if limit, exists := modelLimits[model]; exists {
		// Use 90% of model limit to be safe
		return int(float64(limit) * 0.9)
	}

	// Default safe limit for unknown models
	return 4000
}

// isContextLengthExceededError checks if the error is due to context length exceeded
func (ua *UnifiedAssistant) isContextLengthExceededError(err error) bool {
	errStr := err.Error()
	// Check for common OpenAI context length error patterns
	return strings.Contains(errStr, "context_length_exceeded") ||
		strings.Contains(errStr, "maximum context length") ||
		strings.Contains(errStr, "too many tokens") ||
		strings.Contains(errStr, "token limit exceeded") ||
		strings.Contains(errStr, "context window")
}

// EnableFallbackMode enables graceful degradation mode
func (ua *UnifiedAssistant) EnableFallbackMode() {
	ua.fallbackMode = true
	slog.Info("Fallback mode enabled - using degraded functionality")
}

// DisableFallbackMode disables graceful degradation mode
func (ua *UnifiedAssistant) DisableFallbackMode() {
	ua.fallbackMode = false
	slog.Info("Fallback mode disabled - using full functionality")
}

// generateFallbackTitle generates a simple title when OpenAI is unavailable
func (ua *UnifiedAssistant) generateFallbackTitle(userMessage string) string {
	// Simple fallback: use first few words of user message
	words := strings.Fields(userMessage)
	if len(words) > 5 {
		words = words[:5]
	}
	fallbackTitle := strings.Join(words, " ") + "..."
	return ua.formatTitle(fallbackTitle)
}

// generateFallbackReply generates a simple reply when OpenAI is unavailable
func (ua *UnifiedAssistant) generateFallbackReply(ctx context.Context, conv *model.Conversation) string {
	slog.WarnContext(ctx, "Using fallback reply due to OpenAI unavailability",
		"conversation_id", conv.ID.Hex(),
		"user_id", conv.UserID)

	// Simple fallback responses
	fallbackResponses := []string{
		"I'm currently experiencing technical difficulties. Please try again later.",
		"Sorry, I'm having trouble processing your request right now.",
		"System temporarily unavailable. Please check back in a few minutes.",
		"Unable to generate response at this time. Please try again.",
	}

	// Use conversation ID to deterministically select a response
	hash := int(conv.ID.Hex()[0]) % len(fallbackResponses)
	return fallbackResponses[hash]
}

// executeToolWithFallback executes a tool with graceful degradation
func (ua *UnifiedAssistant) executeToolWithFallback(ctx context.Context, toolName string, arguments string) (string, error) {
	tool := ua.toolRegistry.Get(toolName)
	if tool == nil {
		return "", errors.New("unknown tool: " + toolName)
	}

	// Parse JSON arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", errors.New("failed to parse tool arguments: " + err.Error())
	}

	// Execute the tool with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, args)
	if err != nil {
		slog.WarnContext(ctx, "Tool execution failed, using fallback",
			"tool_name", toolName,
			"error", err)

		// Return informative message instead of error
		return fmt.Sprintf("Unable to execute %s: %s", toolName, err.Error()), nil
	}

	return result, nil
}
