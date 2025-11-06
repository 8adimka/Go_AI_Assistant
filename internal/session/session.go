package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Session represents a user session with conversation context
type Session struct {
	ConversationID string    `json:"conversation_id"`
	Platform       string    `json:"platform"`
	UserID         string    `json:"user_id"`
	ChatID         string    `json:"chat_id"`
	LastActivity   time.Time `json:"last_activity"`
}

// Manager handles session storage and recovery
type Manager struct {
	cache *redisx.Cache
	ttl   time.Duration
	repo  *model.Repository
}

// NewManager creates a new session manager
func NewManager(cache *redisx.Cache, ttl time.Duration, repo *model.Repository) *Manager {
	return &Manager{
		cache: cache,
		ttl:   ttl,
		repo:  repo,
	}
}

// GetSession retrieves a session from Redis or recovers from MongoDB
func (m *Manager) GetSession(ctx context.Context, platform, chatID string) (*Session, error) {
	key := m.generateSessionKey(platform, chatID)
	
	// Try Redis first
	var session Session
	if err := m.cache.Get(ctx, key, &session); err == nil {
		// Update TTL on access (sliding window)
		m.cache.Set(ctx, key, session)
		slog.DebugContext(ctx, "Session found in Redis", 
			"platform", platform, 
			"chat_id", chatID,
			"conversation_id", session.ConversationID)
		return &session, nil
	}
	
	// Redis miss - try MongoDB recovery
	slog.InfoContext(ctx, "Session not found in Redis, attempting MongoDB recovery", 
		"platform", platform, 
		"chat_id", chatID)
	return m.recoverSessionFromMongoDB(ctx, platform, chatID)
}

// SetSession stores a session in Redis
func (m *Manager) SetSession(ctx context.Context, platform, chatID string, session *Session) error {
	key := m.generateSessionKey(platform, chatID)
	return m.cache.Set(ctx, key, session)
}

// DeleteSession removes a session from Redis
func (m *Manager) DeleteSession(ctx context.Context, platform, chatID string) error {
	key := m.generateSessionKey(platform, chatID)
	return m.cache.Delete(ctx, key)
}

// GetOrCreateSession finds an existing session or creates a new one
func (m *Manager) GetOrCreateSession(ctx context.Context, platform, userID, chatID, message string) (string, error) {
	// Try to get existing session
	session, err := m.GetSession(ctx, platform, chatID)
	if err == nil {
		slog.DebugContext(ctx, "Found existing session", 
			"platform", platform, 
			"chat_id", chatID,
			"conversation_id", session.ConversationID)
		return session.ConversationID, nil
	}
	
	// No session found - create a new conversation
	slog.InfoContext(ctx, "Creating new session", 
		"platform", platform, 
		"user_id", userID,
		"chat_id", chatID)
	
	// Create a new conversation
	conversation := &model.Conversation{
		ID:           primitive.NewObjectID(),
		Title:        "Untitled conversation",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Platform:     platform,
		UserID:       userID,
		ChatID:       chatID,
		IsActive:     true,
		LastActivity: time.Now(),
		Messages: []*model.Message{{
			ID:        primitive.NewObjectID(),
			Role:      model.RoleUser,
			Content:   message,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}},
	}
	
	// Generate title and reply would be handled by the assistant later
	// For now, just create the conversation
	
	if err := m.repo.CreateConversation(ctx, conversation); err != nil {
		return "", fmt.Errorf("failed to create conversation: %w", err)
	}
	
	// Create and store session
	newSession := &Session{
		ConversationID: conversation.ID.Hex(),
		Platform:       platform,
		UserID:         userID,
		ChatID:         chatID,
		LastActivity:   time.Now(),
	}
	
	if err := m.SetSession(ctx, platform, chatID, newSession); err != nil {
		slog.WarnContext(ctx, "Failed to store session in Redis", 
			"platform", platform, 
			"chat_id", chatID,
			"error", err)
		// Continue anyway - we have the conversation in MongoDB
	}
	
	slog.InfoContext(ctx, "Created new session", 
		"platform", platform, 
		"chat_id", chatID,
		"conversation_id", conversation.ID.Hex())
	
	return conversation.ID.Hex(), nil
}

// recoverSessionFromMongoDB attempts to recover a session from MongoDB
func (m *Manager) recoverSessionFromMongoDB(ctx context.Context, platform, chatID string) (*Session, error) {
	// Find most recent active conversation for this platform+chatID
	conversations, err := m.repo.FindConversationsByPlatformAndChatID(ctx, platform, chatID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to query conversations for session recovery", 
			"platform", platform, 
			"chat_id", chatID,
			"error", err)
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	
	if len(conversations) == 0 {
		slog.DebugContext(ctx, "No conversations found for session recovery", 
			"platform", platform, 
			"chat_id", chatID)
		return nil, fmt.Errorf("no session found")
	}
	
	// Use most recent active conversation
	latestConv := conversations[0]
	session := &Session{
		ConversationID: latestConv.ID.Hex(),
		Platform:       platform,
		UserID:         latestConv.UserID,
		ChatID:         chatID,
		LastActivity:   time.Now(),
	}
	
	// Restore to Redis
	key := m.generateSessionKey(platform, chatID)
	if err := m.cache.Set(ctx, key, session); err != nil {
		slog.WarnContext(ctx, "Failed to restore session to Redis", 
			"platform", platform, 
			"chat_id", chatID,
			"error", err)
		// Continue anyway - we have the session from MongoDB
	}
	
	slog.InfoContext(ctx, "Session recovered from MongoDB", 
		"platform", platform, 
		"chat_id", chatID,
		"conversation_id", session.ConversationID)
	
	return session, nil
}

// generateSessionKey creates a Redis key for session storage
func (m *Manager) generateSessionKey(platform, chatID string) string {
	return fmt.Sprintf("session:%s:%s", platform, chatID)
}

// SessionMetadata represents the metadata for session-based requests
type SessionMetadata struct {
	Platform string `json:"platform"`
	UserID   string `json:"user_id"`
	ChatID   string `json:"chat_id"`
}

// MarshalJSON custom JSON marshaling for SessionMetadata
func (sm *SessionMetadata) MarshalJSON() ([]byte, error) {
	type Alias SessionMetadata
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(sm),
	})
}

// UnmarshalJSON custom JSON unmarshaling for SessionMetadata
func (sm *SessionMetadata) UnmarshalJSON(data []byte) error {
	type Alias SessionMetadata
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(sm),
	}
	return json.Unmarshal(data, &aux)
}
