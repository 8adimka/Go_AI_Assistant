package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"github.com/8adimka/Go_AI_Assistant/internal/weather"
	ics "github.com/arran4/golang-ical"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli           openai.Client
	cache         *redisx.Cache
	weatherService *weather.FallbackWeatherService
}

func New() *Assistant {
	// Load configuration
	cfg := config.Load()
	redisClient := redisx.MustConnect(cfg.RedisAddr)
	cache := redisx.NewCache(redisClient, 24*time.Hour) // Cache for 24 hours
	
	// Create weather service with fallback
	weatherService := weather.CreateWeatherService(cfg.WeatherApiKey, cache)
	
	return &Assistant{
		cli:           openai.NewClient(),
		cache:         cache,
		weatherService: weatherService,
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

	resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4Turbo, // Faster model for titles
		Messages: msgs,
		MaxTokens: openai.Int(30), // Limit tokens for brevity
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

	for i := 0; i < 15; i++ {
		resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4_1,
			Messages: msgs,
			Tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name:        "get_weather",
					Description: openai.String("Get weather at the given location"),
					Parameters: openai.FunctionParameters{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]string{
								"type": "string",
							},
						},
						"required": []string{"location"},
					},
				}),
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name:        "get_today_date",
					Description: openai.String("Get today's date and time in RFC3339 format"),
				}),
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name:        "get_holidays",
					Description: openai.String("Gets local bank and public holidays. Each line is a single holiday in the format 'YYYY-MM-DD: Holiday Name'."),
					Parameters: openai.FunctionParameters{
						"type": "object",
						"properties": map[string]any{
							"before_date": map[string]string{
								"type":        "string",
								"description": "Optional date in RFC3339 format to get holidays before this date. If not provided, all holidays will be returned.",
							},
							"after_date": map[string]string{
								"type":        "string",
								"description": "Optional date in RFC3339 format to get holidays after this date. If not provided, all holidays will be returned.",
							},
							"max_count": map[string]string{
								"type":        "integer",
								"description": "Optional maximum number of holidays to return. If not provided, all holidays will be returned.",
							},
						},
					},
				}),
			},
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

				switch call.Function.Name {
				case "get_weather":
					var payload struct {
						Location string `json:"location"`
					}
					
					if err := json.Unmarshal([]byte(call.Function.Arguments), &payload); err != nil {
						msgs = append(msgs, openai.ToolMessage("failed to parse location: "+err.Error(), call.ID))
						break
					}
					
					if payload.Location == "" {
						msgs = append(msgs, openai.ToolMessage("location is required", call.ID))
						break
					}
					
					// Get real weather data with fallback
					weatherData, err := a.weatherService.GetCurrentWithFallback(ctx, payload.Location)
					if err != nil {
						slog.ErrorContext(ctx, "Failed to get weather data", "location", payload.Location, "error", err)
						msgs = append(msgs, openai.ToolMessage("weather data unavailable", call.ID))
						break
					}
					
					weatherMessage := weather.FormatWeather(weatherData)
					msgs = append(msgs, openai.ToolMessage(weatherMessage, call.ID))
				case "get_today_date":
					msgs = append(msgs, openai.ToolMessage(time.Now().Format(time.RFC3339), call.ID))
				case "get_holidays":
					link := "https://www.officeholidays.com/ics/spain/catalonia"
					if v := os.Getenv("HOLIDAY_CALENDAR_LINK"); v != "" {
						link = v
					}

					events, err := LoadCalendar(ctx, link)
					if err != nil {
						msgs = append(msgs, openai.ToolMessage("failed to load holiday events", call.ID))
						break
					}

					var payload struct {
						BeforeDate time.Time `json:"before_date,omitempty"`
						AfterDate  time.Time `json:"after_date,omitempty"`
						MaxCount   int       `json:"max_count,omitempty"`
					}

					if err := json.Unmarshal([]byte(call.Function.Arguments), &payload); err != nil {
						msgs = append(msgs, openai.ToolMessage("failed to parse tool call arguments: "+err.Error(), call.ID))
						break
					}

					var holidays []string
					for _, event := range events {
						date, err := event.GetAllDayStartAt()
						if err != nil {
							continue
						}

						if payload.MaxCount > 0 && len(holidays) >= payload.MaxCount {
							break
						}

						if !payload.BeforeDate.IsZero() && date.After(payload.BeforeDate) {
							continue
						}

						if !payload.AfterDate.IsZero() && date.Before(payload.AfterDate) {
							continue
						}

						holidays = append(holidays, date.Format(time.DateOnly)+": "+event.GetProperty(ics.ComponentPropertySummary).Value)
					}

					msgs = append(msgs, openai.ToolMessage(strings.Join(holidays, "\n"), call.ID))
				default:
					return "", errors.New("unknown tool call: " + call.Function.Name)
				}
			}

			continue
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}
