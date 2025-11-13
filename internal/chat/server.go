package chat

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/pb"
	"github.com/8adimka/Go_AI_Assistant/internal/session"
	"github.com/twitchtv/twirp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ pb.ChatService = (*Server)(nil)

type Assistant interface {
	Title(ctx context.Context, conv *model.Conversation) (string, error)
	Reply(ctx context.Context, conv *model.Conversation) (string, error)
}

type Server struct {
	repo           *model.Repository
	assist         Assistant
	sessionManager *session.Manager
}

func NewServer(repo *model.Repository, assist Assistant, sessionManager *session.Manager) *Server {
	return &Server{
		repo:           repo,
		assist:         assist,
		sessionManager: sessionManager,
	}
}

func (s *Server) StartConversation(ctx context.Context, req *pb.StartConversationRequest) (*pb.StartConversationResponse, error) {
	conversation := &model.Conversation{
		ID:           primitive.NewObjectID(),
		Title:        "Untitled conversation",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Platform:     "api", // default for direct API calls
		IsActive:     true,
		LastActivity: time.Now(),
		Messages: []*model.Message{{
			ID:        primitive.NewObjectID(),
			Role:      model.RoleUser,
			Content:   req.GetMessage(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}},
	}

	if strings.TrimSpace(req.GetMessage()) == "" {
		return nil, twirp.RequiredArgumentError("message")
	}

	// choose a title
	title, err := s.assist.Title(ctx, conversation)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to generate conversation title", "error", err)
	} else {
		conversation.Title = title
	}

	// generate a reply
	reply, err := s.assist.Reply(ctx, conversation)
	if err != nil {
		return nil, err
	}

	conversation.Messages = append(conversation.Messages, &model.Message{
		ID:        primitive.NewObjectID(),
		Role:      model.RoleAssistant,
		Content:   reply,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err := s.repo.CreateConversation(ctx, conversation); err != nil {
		return nil, err
	}

	return &pb.StartConversationResponse{
		ConversationId: conversation.ID.Hex(),
		Title:          conversation.Title,
		Reply:          reply,
	}, nil
}

func (s *Server) ContinueConversation(ctx context.Context, req *pb.ContinueConversationRequest) (*pb.ContinueConversationResponse, error) {
	if strings.TrimSpace(req.GetMessage()) == "" {
		return nil, twirp.RequiredArgumentError("message")
	}

	// OPTION 1: Direct conversation_id (existing flow)
	if req.GetConversationId() != "" {
		return s.continueExistingConversation(ctx, req.GetConversationId(), req.GetMessage())
	}

	// OPTION 2: Session-based (new flow) - use session_metadata
	// Extract session metadata from the request
	sessionMetadata := req.GetSessionMetadata()
	if sessionMetadata != nil {
		platform := sessionMetadata.GetPlatform()
		userID := sessionMetadata.GetUserId()
		chatID := sessionMetadata.GetChatId()

		if platform != "" && userID != "" && chatID != "" {
			// Use Session Manager to find or create conversation
			conversationID, err := s.sessionManager.GetOrCreateSession(ctx, platform, userID, chatID, req.GetMessage())
			if err != nil {
				slog.ErrorContext(ctx, "Failed to get or create session",
					"platform", platform, "user_id", userID, "chat_id", chatID, "error", err)
				return nil, twirp.InternalErrorWith(err)
			}

			// Continue with the found/created conversation
			return s.continueExistingConversation(ctx, conversationID, req.GetMessage())
		}
	}

	// If no conversation_id and no valid session_metadata, return error
	return nil, twirp.RequiredArgumentError("conversation_id or session_metadata")
}

// continueExistingConversation handles the actual conversation continuation logic
func (s *Server) continueExistingConversation(ctx context.Context, conversationID, message string) (*pb.ContinueConversationResponse, error) {
	if conversationID == "" {
		// If no conversation ID provided, we need to handle this case
		// For now, we'll return an error, but in production this would create a new conversation
		return nil, twirp.RequiredArgumentError("conversation_id")
	}

	conversation, err := s.repo.DescribeConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Update activity tracking
	conversation.UpdatedAt = time.Now()
	conversation.LastActivity = time.Now()

	// Context management is now handled by the assistant's context manager
	// The assistant will automatically manage token limits and summarization
	slog.DebugContext(ctx, "Context management delegated to assistant",
		"conversation_id", conversation.ID.Hex(),
		"message_count", len(conversation.Messages))

	conversation.Messages = append(conversation.Messages, &model.Message{
		ID:        primitive.NewObjectID(),
		Role:      model.RoleUser,
		Content:   message,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	reply, err := s.assist.Reply(ctx, conversation)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	conversation.Messages = append(conversation.Messages, &model.Message{
		ID:        primitive.NewObjectID(),
		Role:      model.RoleAssistant,
		Content:   reply,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err := s.repo.UpdateConversation(ctx, conversation); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &pb.ContinueConversationResponse{Reply: reply}, nil
}

func (s *Server) ListConversations(ctx context.Context, req *pb.ListConversationsRequest) (*pb.ListConversationsResponse, error) {
	conversations, err := s.repo.ListConversations(ctx)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	resp := &pb.ListConversationsResponse{}
	for _, conv := range conversations {
		conv.Messages = nil // Clear messages to avoid sending large data
		resp.Conversations = append(resp.Conversations, conv.Proto())
	}

	return resp, nil
}

func (s *Server) DescribeConversation(ctx context.Context, req *pb.DescribeConversationRequest) (*pb.DescribeConversationResponse, error) {
	if req.GetConversationId() == "" {
		return nil, twirp.RequiredArgumentError("conversation_id")
	}

	conversation, err := s.repo.DescribeConversation(ctx, req.GetConversationId())
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, twirp.NotFoundError("conversation not found")
	}

	return &pb.DescribeConversationResponse{Conversation: conversation.Proto()}, nil
}

// summarizeConversation is deprecated - context management is now handled by the assistant
// This function is kept for backward compatibility but is no longer used
func (s *Server) summarizeConversation(ctx context.Context, conversation *model.Conversation) string {
	return "" // Context management is now handled by the assistant's context manager
}
