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
	"github.com/8adimka/Go_AI_Assistant/internal/tools/factory"
	"github.com/8adimka/Go_AI_Assistant/internal/tools/registry"
	"github.com/openai/openai-go/v2"
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

	// Use context manager with Redis storage and AI summarization
	contextManager := chat.NewContextManager(
		contextCache,
		maxTokens,
		maxHistory,
		&openAIClient,
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

	// Record OpenAI metrics
	if ua.metrics != nil {
		usage := metrics.TokenUsage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		}
		ua.metrics.RecordOpenAITokens(ctx, "title", string(openai.ChatModelGPT4Turbo),
			conv.UserID, conv.Platform, usage, duration)
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

		// Record OpenAI metrics
		if ua.metrics != nil {
			usage := metrics.TokenUsage{
				PromptTokens:     int(resp.Usage.PromptTokens),
				CompletionTokens: int(resp.Usage.CompletionTokens),
				TotalTokens:      int(resp.Usage.TotalTokens),
			}
			ua.metrics.RecordOpenAITokens(ctx, "reply", string(openai.ChatModelGPT4_1),
				conv.UserID, conv.Platform, usage, duration)
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

// performIntelligentSummarization performs AI-powered summarization to reduce context size
func (ua *UnifiedAssistant) performIntelligentSummarization(ctx context.Context, conversationID string) error {
	slog.InfoContext(ctx, "Performing intelligent summarization",
		"conversation_id", conversationID)

	// Get current context
	currentContext := ua.contextManager.GetContext(conversationID)
	if len(currentContext) <= 2 {
		return errors.New("not enough messages to summarize")
	}

	// Try AI summarization first
	if ua.canUseAISummarization() {
		summary, err := ua.performAISummarization(ctx, currentContext)
		if err == nil {
			slog.InfoContext(ctx, "AI summarization successful",
				"conversation_id", conversationID,
				"summary_length", len(summary))
			return ua.applyAISummary(ctx, conversationID, summary)
		}
		slog.WarnContext(ctx, "AI summarization failed, falling back to basic summarization",
			"conversation_id", conversationID, "error", err)
	}

	// Fallback to basic summarization
	return ua.performBasicSummarization(ctx, conversationID)
}

// canUseAISummarization checks if AI summarization is available
func (ua *UnifiedAssistant) canUseAISummarization() bool {
	return ua.cfg.OpenAIApiKey != ""
}

// performAISummarization creates an AI summary of the conversation
func (ua *UnifiedAssistant) performAISummarization(ctx context.Context, messages []chat.Message) (string, error) {
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
	resp, err := ua.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4Turbo,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant that creates concise summaries of conversations."),
			openai.UserMessage(prompt),
		},
		MaxTokens: openai.Int(200), // Limit summary length
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("empty summary response")
	}

	return resp.Choices[0].Message.Content, nil
}

// applyAISummary applies the AI summary to the conversation context
func (ua *UnifiedAssistant) applyAISummary(ctx context.Context, conversationID string, summary string) error {
	// Clear current context
	ua.contextManager.ClearContext(conversationID)

	// Add summary as system message
	summaryMessage := chat.ConvertModelMessage(&model.Message{
		Role:    model.RoleAssistant, // Use assistant role for summary
		Content: fmt.Sprintf("Previous conversation summary: %s", summary),
	})
	if err := ua.contextManager.AddMessage(ctx, conversationID, summaryMessage); err != nil {
		return fmt.Errorf("failed to add summary message: %w", err)
	}

	// Keep only the most recent messages (2-3 messages)
	currentContext := ua.contextManager.GetContext(conversationID)
	keepCount := 3
	if len(currentContext) < keepCount {
		keepCount = len(currentContext)
	}

	for i := len(currentContext) - keepCount; i < len(currentContext); i++ {
		if err := ua.contextManager.AddMessage(ctx, conversationID, currentContext[i]); err != nil {
			slog.WarnContext(ctx, "Failed to add message during AI summarization",
				"conversation_id", conversationID, "error", err)
		}
	}

	return nil
}

// performBasicSummarization performs basic summarization without AI
func (ua *UnifiedAssistant) performBasicSummarization(ctx context.Context, conversationID string) error {
	slog.InfoContext(ctx, "Performing basic summarization",
		"conversation_id", conversationID)

	// Get current context
	currentContext := ua.contextManager.GetContext(conversationID)
	if len(currentContext) <= 2 {
		return errors.New("not enough messages to summarize")
	}

	// Keep only the most recent messages (emergency fallback)
	keepCount := 3 // Keep last 3 messages
	if len(currentContext) < keepCount {
		keepCount = len(currentContext)
	}

	// Clear current context
	ua.contextManager.ClearContext(conversationID)

	// Add back only the most recent messages
	for i := len(currentContext) - keepCount; i < len(currentContext); i++ {
		if err := ua.contextManager.AddMessage(ctx, conversationID, currentContext[i]); err != nil {
			slog.WarnContext(ctx, "Failed to add message during basic summarization",
				"conversation_id", conversationID, "error", err)
		}
	}

	slog.InfoContext(ctx, "Basic summarization completed",
		"conversation_id", conversationID,
		"original_messages", len(currentContext),
		"kept_messages", keepCount)

	return nil
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
func (ua *UnifiedAssistant) convertToolsToOpenAIFormat() []openai.ChatCompletionToolUnionParam {
	var tools []openai.ChatCompletionToolUnionParam

	for _, tool := range ua.toolRegistry.GetAll() {
		tools = append(tools, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        tool.Name(),
			Description: openai.String(tool.Description()),
			Parameters:  openai.FunctionParameters(tool.Parameters()),
		}))
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
func (ua *UnifiedAssistant) estimateTokenCount(msgs []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolUnionParam) int {
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
