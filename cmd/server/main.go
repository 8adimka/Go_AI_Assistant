package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/assistant"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/health"
	"github.com/8adimka/Go_AI_Assistant/internal/httpx"
	"github.com/8adimka/Go_AI_Assistant/internal/logging"
	"github.com/8adimka/Go_AI_Assistant/internal/metrics"
	"github.com/8adimka/Go_AI_Assistant/internal/mongox"
	"github.com/8adimka/Go_AI_Assistant/internal/otel"
	"github.com/8adimka/Go_AI_Assistant/internal/pb"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"github.com/8adimka/Go_AI_Assistant/internal/session"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/twitchtv/twirp"
)

func main() {
	ctx := context.Background()

	// Load configuration from .env file
	cfg := config.Load()

	// Initialize secure logger
	secureLogger := logging.NewSecureLogger(slog.Default())

	// Log configuration safely
	secureLogger.Info("Configuration loaded", "config", cfg.SafeString())

	// Initialize OpenTelemetry
	shutdown, err := otel.InitOpenTelemetry(ctx, "go-ai-assistant")
	if err != nil {
		secureLogger.Error("Failed to initialize OpenTelemetry", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	// Set OpenAI API key for the assistant
	os.Setenv("OPENAI_API_KEY", cfg.OpenAIApiKey)

	// Connect to MongoDB
	mongo := mongox.MustConnect(cfg.MongoURI, "acai")

	// Connect to Redis
	redisClient := redisx.MustConnect(cfg.RedisAddr)

	// Initialize metrics
	meter := otel.GetMeter()
	appMetrics, err := metrics.NewMetrics(meter)
	if err != nil {
		secureLogger.Error("Failed to initialize metrics", "error", err)
		os.Exit(1)
	}

	repo := model.New(mongo)
	assist := assistant.New(appMetrics)

	// Create Redis cache for session management with configurable TTL
	sessionTTL := time.Duration(cfg.SessionTTLMinutes) * time.Minute
	redisCache := redisx.NewCache(redisClient, sessionTTL)

	// Create session manager
	sessionManager := session.NewManager(redisCache, sessionTTL, repo)

	server := chat.NewServer(repo, assist, sessionManager)

	// Initialize rate limiter with configuration
	rateLimiter := httpx.NewRateLimiter(cfg.APIRateLimitRPS, cfg.APIRateLimitBurst)

	// Configure handler
	handler := mux.NewRouter()
	handler.Use(
		rateLimiter.Middleware(), // Rate limiting first!
		appMetrics.HTTPMetricsMiddleware(),
		httpx.OTelMiddleware(),
		httpx.Logger(),
		httpx.Recovery(),
	)

	// Health checks
	healthChecker := health.NewHealthChecker(mongo.Client(), redisClient)
	handler.HandleFunc("/health", healthChecker.HealthHandler)
	handler.HandleFunc("/ready", healthChecker.ReadyHandler)

	// Metrics endpoint - Prometheus metrics (always available, protected with API key)
	auth := httpx.NewAPIKeyAuth(cfg.APIKey)
	handler.Handle("/metrics", auth.Middleware()(promhttp.Handler()))

	if cfg.APIKey == "" || cfg.APIKey == "changeme_in_production" {
		secureLogger.Warn("API_KEY is not set or using default value - metrics endpoint is accessible but requires authentication")
	} else {
		secureLogger.Info("Metrics endpoint protected with API key")
	}

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Clippy!")
	})

	// Test endpoint to verify our code is running
	handler.HandleFunc("/test-docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<h1>Test Documentation</h1><p>This endpoint works!</p>")
	})

	handler.PathPrefix("/twirp/").Handler(pb.NewChatServiceServer(server, twirp.WithServerJSONSkipDefaults(true)))

	// Serve swagger.json file for Swagger UI - always return full documentation
	handler.HandleFunc("/docs/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"swagger": "2.0",
			"info": {
				"title": "Go AI Assistant API",
				"version": "1.0",
				"description": "Production-ready AI assistant backend with modular tools, Redis caching, and comprehensive monitoring"
			},
			"host": "localhost:8080",
			"basePath": "/",
			"paths": {
				"/": {
					"get": {
						"description": "Get basic service information",
						"produces": ["text/plain"],
						"tags": ["system"],
						"summary": "Service information",
						"responses": {
							"200": {
								"description": "Service information",
								"schema": {"type": "string"}
							}
						}
					}
				},
				"/health": {
					"get": {
						"description": "Check service health status including MongoDB and Redis connectivity",
						"produces": ["application/json"],
						"tags": ["system"],
						"summary": "Health check",
						"responses": {
							"200": {
								"description": "OK",
								"schema": {"$ref": "#/definitions/HealthResponse"}
							}
						}
					}
				},
				"/ready": {
					"get": {
						"description": "Check service readiness for traffic",
						"produces": ["application/json"],
						"tags": ["system"],
						"summary": "Readiness check",
						"responses": {
							"200": {
								"description": "OK",
								"schema": {"$ref": "#/definitions/HealthResponse"}
							}
						}
					}
				},
				"/metrics": {
					"get": {
						"security": [{"ApiKeyAuth": []}],
						"description": "Get Prometheus metrics for monitoring (requires API key)",
						"produces": ["text/plain"],
						"tags": ["system"],
						"summary": "Prometheus metrics",
						"responses": {
							"200": {
								"description": "Prometheus metrics",
								"schema": {"type": "string"}
							},
							"401": {
								"description": "Unauthorized",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							}
						}
					}
				},
				"/twirp/chat.ChatService/StartConversation": {
					"post": {
						"description": "Create a new conversation with the AI assistant. The assistant can answer questions, provide weather information, date/time, and holiday information.",
						"consumes": ["application/json"],
						"produces": ["application/json"],
						"tags": ["conversations"],
						"summary": "Start a new conversation",
						"parameters": [
							{
								"description": "Start conversation request",
								"name": "request",
								"in": "body",
								"required": true,
								"schema": {"$ref": "#/definitions/StartConversationRequest"}
							}
						],
						"responses": {
							"200": {
								"description": "OK",
								"schema": {"$ref": "#/definitions/StartConversationResponse"}
							},
							"400": {
								"description": "Bad Request",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							},
							"500": {
								"description": "Internal Server Error",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							}
						}
					}
				},
				"/twirp/chat.ChatService/ContinueConversation": {
					"post": {
						"description": "Continue an existing conversation with the AI assistant. Supports both direct conversation_id and session-based conversations for stateless clients.",
						"consumes": ["application/json"],
						"produces": ["application/json"],
						"tags": ["conversations"],
						"summary": "Continue an existing conversation",
						"parameters": [
							{
								"description": "Continue conversation request",
								"name": "request",
								"in": "body",
								"required": true,
								"schema": {"$ref": "#/definitions/ContinueConversationRequest"}
							}
						],
						"responses": {
							"200": {
								"description": "OK",
								"schema": {"$ref": "#/definitions/ContinueConversationResponse"}
							},
							"400": {
								"description": "Bad Request",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							},
							"404": {
								"description": "Not Found",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							},
							"500": {
								"description": "Internal Server Error",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							}
						}
					}
				},
				"/twirp/chat.ChatService/ListConversations": {
					"post": {
						"description": "Get list of recent conversations. Messages are excluded from the response to avoid large payloads.",
						"consumes": ["application/json"],
						"produces": ["application/json"],
						"tags": ["conversations"],
						"summary": "List conversations",
						"responses": {
							"200": {
								"description": "OK",
								"schema": {"$ref": "#/definitions/ListConversationsResponse"}
							},
							"500": {
								"description": "Internal Server Error",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							}
						}
					}
				},
				"/twirp/chat.ChatService/DescribeConversation": {
					"post": {
						"description": "Get detailed information about a specific conversation including all messages.",
						"consumes": ["application/json"],
						"produces": ["application/json"],
						"tags": ["conversations"],
						"summary": "Get conversation details",
						"parameters": [
							{
								"description": "Describe conversation request",
								"name": "request",
								"in": "body",
								"required": true,
								"schema": {"$ref": "#/definitions/DescribeConversationRequest"}
							}
						],
						"responses": {
							"200": {
								"description": "OK",
								"schema": {"$ref": "#/definitions/DescribeConversationResponse"}
							},
							"400": {
								"description": "Bad Request",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							},
							"404": {
								"description": "Not Found",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							},
							"500": {
								"description": "Internal Server Error",
								"schema": {"$ref": "#/definitions/ErrorResponse"}
							}
						}
					}
				}
			},
			"definitions": {
				"HealthResponse": {
					"type": "object",
					"properties": {
						"status": {"type": "string", "example": "healthy"},
						"checks": {
							"type": "object",
							"additionalProperties": {"type": "string"},
							"example": {"mongodb": "ok", "redis": "ok"}
						}
					}
				},
				"ErrorResponse": {
					"type": "object",
					"properties": {
						"code": {"type": "integer", "example": 400},
						"message": {"type": "string", "example": "Bad Request"},
						"details": {"type": "string", "example": "Missing required field: message"}
					}
				},
				"StartConversationRequest": {
					"type": "object",
					"properties": {
						"message": {"type": "string", "example": "What's the weather in Barcelona?"},
						"session_metadata": {"$ref": "#/definitions/SessionMetadata"}
					}
				},
				"StartConversationResponse": {
					"type": "object",
					"properties": {
						"conversation_id": {"type": "string", "example": "507f1f77bcf86cd799439011"},
						"title": {"type": "string", "example": "Weather in Barcelona"},
						"reply": {"type": "string", "example": "The weather in Barcelona is sunny with 22°C..."}
					}
				},
				"ContinueConversationRequest": {
					"type": "object",
					"properties": {
						"conversation_id": {"type": "string", "example": "507f1f77bcf86cd799439011"},
						"message": {"type": "string", "example": "What about tomorrow?"},
						"session_metadata": {"$ref": "#/definitions/SessionMetadata"}
					}
				},
				"ContinueConversationResponse": {
					"type": "object",
					"properties": {
						"reply": {"type": "string", "example": "Tomorrow will be partly cloudy with 20°C..."}
					}
				},
				"ListConversationsResponse": {
					"type": "object",
					"properties": {
						"conversations": {
							"type": "array",
							"items": {"$ref": "#/definitions/Conversation"}
						}
					}
				},
				"DescribeConversationRequest": {
					"type": "object",
					"properties": {
						"conversation_id": {"type": "string", "example": "507f1f77bcf86cd799439011"}
					}
				},
				"DescribeConversationResponse": {
					"type": "object",
					"properties": {
						"conversation": {"$ref": "#/definitions/Conversation"}
					}
				},
				"Conversation": {
					"type": "object",
					"properties": {
						"id": {"type": "string", "example": "507f1f77bcf86cd799439011"},
						"title": {"type": "string", "example": "Weather discussion"},
						"timestamp": {"type": "string", "example": "2025-11-07T20:15:00Z"},
						"messages": {
							"type": "array",
							"items": {"$ref": "#/definitions/Message"}
						}
					}
				},
				"Message": {
					"type": "object",
					"properties": {
						"id": {"type": "string", "example": "507f1f77bcf86cd799439012"},
						"role": {"type": "string", "example": "user"},
						"content": {"type": "string", "example": "What's the weather like?"},
						"timestamp": {"type": "string", "example": "2025-11-07T20:15:00Z"}
					}
				},
				"SessionMetadata": {
					"type": "object",
					"properties": {
						"platform": {"type": "string", "example": "telegram"},
						"user_id": {"type": "string", "example": "12345"},
						"chat_id": {"type": "string", "example": "67890"}
					}
				}
			},
			"securityDefinitions": {
				"ApiKeyAuth": {
					"type": "apiKey",
					"name": "X-API-Key",
					"in": "header"
				}
			}
		}`)
	})

	// Swagger documentation
	handler.PathPrefix("/docs/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"), // Use explicit URL to doc.json
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Static documentation - serve HTML content directly
	handler.HandleFunc("/api-docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go AI Assistant API Documentation</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 2rem; border-radius: 10px; margin-bottom: 2rem; }
        .endpoint { background: #f8f9fa; border-left: 4px solid #667eea; padding: 1.5rem; margin: 1rem 0; border-radius: 5px; }
        .method { display: inline-block; background: #667eea; color: white; padding: 0.3rem 0.8rem; border-radius: 4px; font-weight: bold; margin-right: 1rem; }
        .path { font-family: 'Courier New', monospace; font-weight: bold; color: #495057; }
        .description { margin: 1rem 0; color: #6c757d; }
        .example { background: #e9ecef; padding: 1rem; border-radius: 5px; margin: 1rem 0; font-family: 'Courier New', monospace; font-size: 0.9rem; }
        .section { margin: 2rem 0; }
        .section-title { border-bottom: 2px solid #667eea; padding-bottom: 0.5rem; margin-bottom: 1rem; }
        .tag { background: #17a2b8; color: white; padding: 0.2rem 0.5rem; border-radius: 3px; font-size: 0.8rem; margin-left: 1rem; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🤖 Go AI Assistant API</h1>
        <p>Production-ready AI assistant backend with modular tools, Redis caching, and comprehensive monitoring</p>
        <p><strong>Version:</strong> 1.0 | <strong>Base URL:</strong> http://localhost:8080</p>
    </div>

    <div class="section">
        <h2 class="section-title">📋 Overview</h2>
        <p>The Go AI Assistant provides a powerful API for interacting with AI assistants that can:</p>
        <ul>
            <li>Answer questions using OpenAI GPT models</li>
            <li>Provide real-time weather information</li>
            <li>Get current date/time and holiday information</li>
            <li>Maintain conversation context with session management</li>
            <li>Support stateless clients (Telegram, Web, Mobile)</li>
        </ul>
    </div>

    <div class="section">
        <h2 class="section-title">💬 Conversation Endpoints</h2>

        <div class="endpoint">
            <div class="method">POST</div>
            <span class="path">/twirp/chat.ChatService/StartConversation</span>
            <span class="tag">conversations</span>
            <div class="description">Start a new conversation with the AI assistant</div>
            <div class="example">
                <strong>Request:</strong><br>
                {<br>
                &nbsp;&nbsp;"message": "What's the weather in Barcelona?"<br>
                }<br><br>
                <strong>Response:</strong><br>
                {<br>
                &nbsp;&nbsp;"conversation_id": "507f1f77bcf86cd799439011",<br>
                &nbsp;&nbsp;"title": "Weather in Barcelona",<br>
                &nbsp;&nbsp;"reply": "The weather in Barcelona is sunny with 22°C..."<br>
                }
            </div>
        </div>

        <div class="endpoint">
            <div class="method">POST</div>
            <span class="path">/twirp/chat.ChatService/ContinueConversation</span>
            <span class="tag">conversations</span>
            <div class="description">Continue an existing conversation. Supports both direct conversation_id and session-based conversations.</div>
            <div class="example">
                <strong>Request (with conversation_id):</strong><br>
                {<br>
                &nbsp;&nbsp;"conversation_id": "507f1f77bcf86cd799439011",<br>
                &nbsp;&nbsp;"message": "What about tomorrow?"<br>
                }<br><br>
                <strong>Request (with session):</strong><br>
                {<br>
                &nbsp;&nbsp;"message": "What about tomorrow?",<br>
                &nbsp;&nbsp;"session_metadata": {<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"platform": "telegram",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"user_id": "12345",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"chat_id": "67890"<br>
                &nbsp;&nbsp;}<br>
                }<br><br>
                <strong>Response:</strong><br>
                {<br>
                &nbsp;&nbsp;"reply": "Tomorrow will be partly cloudy with 20°C..."<br>
                }
            </div>
        </div>

        <div class="endpoint">
            <div class="method">POST</div>
            <span class="path">/twirp/chat.ChatService/ListConversations</span>
            <span class="tag">conversations</span>
            <div class="description">Get list of recent conversations (messages excluded to avoid large payloads)</div>
            <div class="example">
                <strong>Request:</strong><br>
                {}<br><br>
                <strong>Response:</strong><br>
                {<br>
                &nbsp;&nbsp;"conversations": [<br>
                &nbsp;&nbsp;&nbsp;&nbsp;{<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"id": "507f1f77bcf86cd799439011",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"title": "Weather discussion",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"timestamp": "2025-11-07T20:15:00Z"<br>
                &nbsp;&nbsp;&nbsp;&nbsp;}<br>
                &nbsp;&nbsp;]<br>
                }
            </div>
        </div>

        <div class="endpoint">
            <div class="method">POST</div>
            <span class="path">/twirp/chat.ChatService/DescribeConversation</span>
            <span class="tag">conversations</span>
            <div class="description">Get detailed information about a specific conversation including all messages</div>
            <div class="example">
                <strong>Request:</strong><br>
                {<br>
                &nbsp;&nbsp;"conversation_id": "507f1f77bcf86cd799439011"<br>
                }<br><br>
                <strong>Response:</strong><br>
                {<br>
                &nbsp;&nbsp;"conversation": {<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"id": "507f1f77bcf86cd799439011",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"title": "Weather discussion",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"timestamp": "2025-11-07T20:15:00Z",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"messages": [<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;{<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"id": "507f1f77bcf86cd799439012",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"role": "user",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"content": "What's the weather like?",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"timestamp": "2025-11-07T20:15:00Z"<br>
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;}<br>
                &nbsp;&nbsp;&nbsp;&nbsp;]<br>
                &nbsp;&nbsp;}<br>
                }
            </div>
        </div>
    </div>

    <div class="section">
        <h2 class="section-title">⚙️ System Endpoints</h2>

        <div class="endpoint">
            <div class="method">GET</div>
            <span class="path">/health</span>
            <span class="tag">system</span>
            <div class="description">Health check including MongoDB and Redis connectivity</div>
            <div class="example">
                <strong>Response:</strong><br>
                {<br>
                &nbsp;&nbsp;"status": "healthy",<br>
                &nbsp;&nbsp;"checks": {<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"mongodb": "ok",<br>
                &nbsp;&nbsp;&nbsp;&nbsp;"redis": "ok"<br>
                &nbsp;&nbsp;}<br>
                }
            </div>
        </div>

        <div class="endpoint">
            <div class="method">GET</div>
            <span class="path">/ready</span>
            <span class="tag">system</span>
            <div class="description">Readiness check for traffic</div>
            <div class="example">
                <strong>Response:</strong><br>
                {<br>
                &nbsp;&nbsp;"status": "ready"<br>
                }
            </div>
        </div>

        <div class="endpoint">
            <div class="method">GET</div>
            <span class="path">/metrics</span>
            <span class="tag">system</span>
            <div class="description">Prometheus metrics for monitoring (requires API key)</div>
            <div class="example">
                <strong>Headers:</strong><br>
                X-API-Key: your-api-key-here<br><br>
                <strong>Response:</strong> Prometheus metrics format
            </div>
        </div>

        <div class="endpoint">
            <div class="method">GET</div>
            <span class="path">/</span>
            <span class="tag">system</span>
            <div class="description">Basic service information</div>
            <div class="example">
                <strong>Response:</strong><br>
                Hi, my name is Clippy!
            </div>
        </div>
    </div>

    <div class="section">
        <h2 class="section-title">🛠️ Available Tools</h2>
        <ul>
            <li><strong>get_weather</strong> - Get current weather information for any location</li>
            <li><strong>get_today_date</strong> - Get current date and time information</li>
            <li><strong>get_holidays</strong> - Get holiday information for different regions</li>
        </ul>
    </div>

    <div class="section">
        <h2 class="section-title">🔧 Quick Start</h2>
        <div class="example">
            # Start services<br>
            docker-compose up -d<br>
            make migrate-up<br><br>
            # Start server<br>
            go run ./cmd/server<br><br>
            # Test API<br>
            curl -X POST http://localhost:8080/twirp/chat.ChatService/StartConversation \<br>
            -H "Content-Type: application/json" \<br>
            -d '{"message": "Hello!"}'
        </div>
    </div>

    <div class="section">
        <h2 class="section-title">📚 Additional Resources</h2>
        <ul>
            <li><a href="https://github.com/8adimka/Go_AI_Assistant">GitHub Repository</a></li>
            <li><a href="/docs/">Swagger UI</a> (if available)</li>
            <li><a href="ARCHITECTURE.md">Architecture Documentation</a></li>
            <li><a href="PRODUCTION_READINESS.md">Production Readiness Checklist</a></li>
        </ul>
    </div>
</body>
</html>`
		fmt.Fprint(w, html)
	})

	// Start the server with graceful shutdown
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		secureLogger.Info("Starting the server...", "port", "8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			secureLogger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	secureLogger.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		secureLogger.Error("Server forced to shutdown", "error", err)
	}

	secureLogger.Info("Server exited")
}
