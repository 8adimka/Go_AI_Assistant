package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"github.com/8adimka/Go_AI_Assistant/internal/retry"
	"github.com/8adimka/Go_AI_Assistant/internal/tools/factory"
	"github.com/8adimka/Go_AI_Assistant/internal/tools/registry"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli           openai.Client
	cache         *redisx.Cache
	toolRegistry  *registry.ToolRegistry
	retryConfig   retry.RetryConfig
}

func New() *Assistant {
	// Load configuration
	cfg := config.Load()
	redisClient := redisx.MustConnect(cfg.RedisAddr)
	cache := redisx.NewCache(redisClient, 24*time.Hour) // Cache for 24 hours
	
	// Create tool registry with all available tools
	toolFactory := factory.NewFactory(cfg)
	toolRegistry := toolFactory.CreateAllTools()
	
	return &Assistant{
		cli:           openai.NewClient(),
		cache:         cache,
		toolRegistry:  toolRegistry,
		retryConfig:   retry.ConfigFromAppConfig(cfg),
	}
}

func (a *Assistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "An empty conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation", "conversation_id", conv.ID)

	// Try to get from cache first
	userMessage := conv.Messages[0].Content
	cacheKey := a.cache.GenerateKey("title", userMessage)
	
	var cachedTitle string
	if err := a.cache.Get(ctx, cacheKey, &cachedTitle); err == nil {
		slog.InfoContext(ctx, "Title retrieved from cache", "conversation_id", conv.ID)
		return cachedTitle, nil
	} else if !errors.Is(err, redisx.ErrCacheMiss) {
		slog.WarnContext(ctx, "Cache error, proceeding without cache", "error", err)
	}

	// Improved prompt for title generation
	titlePrompt := `Generate a very concise and descriptive title for this conversation. 
The title should:
- Be 3-7 words maximum
- Focus on the main topic or question
- Be in title case (capitalize main words)
- Avoid answering the question, just describe the topic
- No special characters, emojis, or punctuation at the end
- Maximum 60 characters

Examples:
- User: "What's the weather in Barcelona?" → "Weather in Barcelona"
- User: "Tell me about machine learning" → "Machine Learning Overview"
- User: "How to cook pasta carbonara" → "Pasta Carbonara Recipe"

Generate title for:`

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(titlePrompt),
		openai.UserMessage(userMessage),
	}

	// Use retry logic for OpenAI API call
	resp, err := retry.RetryWithResult(ctx, a.retryConfig, func() (*openai.ChatCompletion, error) {
		return a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4Turbo, // Faster model for titles
			Messages: msgs,
			MaxTokens: openai.Int(30), // Limit tokens for brevity
		})
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	title := resp.Choices[0].Message.Content
	title = a.formatTitle(title)

	// Save to cache
	if err := a.cache.Set(ctx, cacheKey, title); err != nil {
		slog.WarnContext(ctx, "Failed to cache title", "error", err)
	}

	return title, nil
}

// formatTitle formats and validates the title
func (a *Assistant) formatTitle(title string) string {
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
	title = a.toTitleCase(title)
	
	return title
}

// toTitleCase converts string to Title Case
func (a *Assistant) toTitleCase(s string) string {
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

func (a *Assistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation", "conversation_id", conv.ID)

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful, concise AI assistant. Provide accurate, safe, and clear responses."),
	}

	for _, m := range conv.Messages {
		switch m.Role {
		case model.RoleUser:
			msgs = append(msgs, openai.UserMessage(m.Content))
		case model.RoleAssistant:
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}

	// Convert registered tools to OpenAI tool format
	tools := a.convertToolsToOpenAIFormat()

	for i := 0; i < 15; i++ {
		// Use retry logic for OpenAI API call
		resp, err := retry.RetryWithResult(ctx, a.retryConfig, func() (*openai.ChatCompletion, error) {
			return a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
				Model:    openai.ChatModelGPT4_1,
				Messages: msgs,
				Tools:    tools,
			})
		})

		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		if message := resp.Choices[0].Message; len(message.ToolCalls) > 0 {
			msgs = append(msgs, message.ToParam())

			for _, call := range message.ToolCalls {
				slog.InfoContext(ctx, "Tool call received", "name", call.Function.Name, "args", call.Function.Arguments)

				// Execute tool using the registry
				result, err := a.executeTool(ctx, call.Function.Name, call.Function.Arguments)
				if err != nil {
					slog.ErrorContext(ctx, "Tool execution failed", "name", call.Function.Name, "error", err)
					msgs = append(msgs, openai.ToolMessage("tool execution failed: "+err.Error(), call.ID))
				} else {
					msgs = append(msgs, openai.ToolMessage(result, call.ID))
				}
			}

			continue
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}

// convertToolsToOpenAIFormat converts registered tools to OpenAI tool format
func (a *Assistant) convertToolsToOpenAIFormat() []openai.ChatCompletionToolUnionParam {
	var tools []openai.ChatCompletionToolUnionParam

	for _, tool := range a.toolRegistry.GetAll() {
		tools = append(tools, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        tool.Name(),
			Description: openai.String(tool.Description()),
			Parameters:  openai.FunctionParameters(tool.Parameters()),
		}))
	}

	return tools
}

// executeTool executes a tool by name with the provided arguments
func (a *Assistant) executeTool(ctx context.Context, toolName string, arguments string) (string, error) {
	tool := a.toolRegistry.Get(toolName)
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
