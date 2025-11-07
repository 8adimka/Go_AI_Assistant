package main

// @title Go AI Assistant API
// @version 1.0
// @description Production-ready AI assistant backend with modular tools, Redis caching, and comprehensive monitoring
// @contact.name API Support
// @contact.url https://github.com/8adimka/Go_AI_Assistant
// @license.name MIT
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// StartConversationRequest represents request to start a new conversation
type StartConversationRequest struct {
	Message         string           `json:"message" example:"What's the weather in Barcelona?"`
	SessionMetadata *SessionMetadata `json:"session_metadata,omitempty"`
}

// StartConversationResponse represents response from starting a conversation
type StartConversationResponse struct {
	ConversationID string `json:"conversation_id" example:"507f1f77bcf86cd799439011"`
	Title          string `json:"title" example:"Weather in Barcelona"`
	Reply          string `json:"reply" example:"The weather in Barcelona is sunny with 22°C..."`
}

// ContinueConversationRequest represents request to continue a conversation
type ContinueConversationRequest struct {
	ConversationID  string           `json:"conversation_id,omitempty" example:"507f1f77bcf86cd799439011"`
	Message         string           `json:"message" example:"What about tomorrow?"`
	SessionMetadata *SessionMetadata `json:"session_metadata,omitempty"`
}

// ContinueConversationResponse represents response from continuing a conversation
type ContinueConversationResponse struct {
	Reply string `json:"reply" example:"Tomorrow will be partly cloudy with 20°C..."`
}

// ListConversationsResponse represents response from listing conversations
type ListConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
}

// DescribeConversationRequest represents request to describe a conversation
type DescribeConversationRequest struct {
	ConversationID string `json:"conversation_id" example:"507f1f77bcf86cd799439011"`
}

// DescribeConversationResponse represents response from describing a conversation
type DescribeConversationResponse struct {
	Conversation Conversation `json:"conversation"`
}

// SessionMetadata represents session information for stateless clients
type SessionMetadata struct {
	Platform string `json:"platform" example:"telegram"`
	UserID   string `json:"user_id" example:"12345"`
	ChatID   string `json:"chat_id" example:"67890"`
}

// Conversation represents a conversation with the AI assistant
type Conversation struct {
	ID        string    `json:"id" example:"507f1f77bcf86cd799439011"`
	Title     string    `json:"title" example:"Weather discussion"`
	Timestamp string    `json:"timestamp" example:"2025-11-07T20:15:00Z"`
	Messages  []Message `json:"messages,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	ID        string `json:"id" example:"507f1f77bcf86cd799439012"`
	Role      string `json:"role" example:"user"`
	Content   string `json:"content" example:"What's the weather like?"`
	Timestamp string `json:"timestamp" example:"2025-11-07T20:15:00Z"`
}

// @Summary Start a new conversation
// @Description Create a new conversation with the AI assistant. The assistant can answer questions, provide weather information, date/time, and holiday information.
// @Tags conversations
// @Accept json
// @Produce json
// @Param request body StartConversationRequest true "Start conversation request"
// @Success 200 {object} StartConversationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /twirp/chat.ChatService/StartConversation [post]
func _startConversation() {}

// @Summary Continue an existing conversation
// @Description Continue an existing conversation with the AI assistant. Supports both direct conversation_id and session-based conversations for stateless clients.
// @Tags conversations
// @Accept json
// @Produce json
// @Param request body ContinueConversationRequest true "Continue conversation request"
// @Success 200 {object} ContinueConversationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /twirp/chat.ChatService/ContinueConversation [post]
func _continueConversation() {}

// @Summary List conversations
// @Description Get list of recent conversations. Messages are excluded from the response to avoid large payloads.
// @Tags conversations
// @Accept json
// @Produce json
// @Success 200 {object} ListConversationsResponse
// @Failure 500 {object} ErrorResponse
// @Router /twirp/chat.ChatService/ListConversations [post]
func _listConversations() {}

// @Summary Get conversation details
// @Description Get detailed information about a specific conversation including all messages.
// @Tags conversations
// @Accept json
// @Produce json
// @Param request body DescribeConversationRequest true "Describe conversation request"
// @Success 200 {object} DescribeConversationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /twirp/chat.ChatService/DescribeConversation [post]
func _describeConversation() {}

// @Summary Health check
// @Description Check service health status including MongoDB and Redis connectivity
// @Tags system
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func _health() {}

// @Summary Readiness check
// @Description Check service readiness for traffic
// @Tags system
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /ready [get]
func _ready() {}

// @Summary Prometheus metrics
// @Description Get Prometheus metrics for monitoring (requires API key)
// @Tags system
// @Produce text/plain
// @Security ApiKeyAuth
// @Success 200 {string} string "Prometheus metrics"
// @Failure 401 {object} ErrorResponse
// @Router /metrics [get]
func _metrics() {}

// @Summary Service information
// @Description Get basic service information
// @Tags system
// @Produce plain
// @Success 200 {string} string "Service information"
// @Router / [get]
func _root() {}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"Bad Request"`
	Details string `json:"details,omitempty" example:"Missing required field: message"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status string            `json:"status" example:"healthy"`
	Checks map[string]string `json:"checks" example:"mongodb:ok,redis:ok"`
}
