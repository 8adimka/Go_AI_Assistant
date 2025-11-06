package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/8adimka/Go_AI_Assistant/internal/chat"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/pb"
	"github.com/twitchtv/twirp"
)

// MockAssistant is a mock implementation of the Assistant interface for testing
type MockAssistant struct {
	TitleResponse string
	ReplyResponse string
	TitleError    error
	ReplyError    error
}

func (m *MockAssistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if m.TitleError != nil {
		return "", m.TitleError
	}
	return m.TitleResponse, nil
}

func (m *MockAssistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if m.ReplyError != nil {
		return "", m.ReplyError
	}
	return m.ReplyResponse, nil
}

func TestServer_InputValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for empty message", func(t *testing.T) {
		// Use nil repository - this will cause errors when trying to save, but we're testing validation first
		mockAssist := &MockAssistant{
			TitleResponse: "Test Title",
			ReplyResponse: "Test Reply",
		}
		srv := chat.NewServer(nil, mockAssist, nil)

		// Try to start conversation with empty message
		_, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "",
		})

		// Assert error is returned
		if err == nil {
			t.Fatal("expected error for empty message, got nil")
		}

		// Assert it's a required argument error
		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.InvalidArgument {
			t.Errorf("expected twirp.InvalidArgument error, got %v", err)
		}
	})

	t.Run("handles whitespace-only message as empty", func(t *testing.T) {
		// Use nil repository
		mockAssist := &MockAssistant{
			TitleResponse: "Test Title",
			ReplyResponse: "Test Reply",
		}
		srv := chat.NewServer(nil, mockAssist, nil)

		// Try to start conversation with whitespace-only message
		_, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "   \t\n  ",
		})

		// Assert error is returned
		if err == nil {
			t.Fatal("expected error for whitespace-only message, got nil")
		}

		// Assert it's a required argument error
		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.InvalidArgument {
			t.Errorf("expected twirp.InvalidArgument error, got %v", err)
		}
	})

	t.Run("returns error when reply generation fails", func(t *testing.T) {
		// Use nil repository
		// Create mock assistant that fails on reply generation
		mockAssist := &MockAssistant{
			TitleResponse: "Weather in Barcelona",
			ReplyError:    twirp.InternalError("reply generation failed"),
		}
		srv := chat.NewServer(nil, mockAssist, nil)

		// Start conversation should fail if reply fails
		_, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "What is the weather like in Barcelona?",
		})

		// Assert error is returned
		if err == nil {
			t.Fatal("expected error for reply generation failure, got nil")
		}

		// Verify error contains the expected message
		if !strings.Contains(err.Error(), "reply generation failed") {
			t.Errorf("expected error to contain 'reply generation failed', got %v", err)
		}
	})
}

func TestServer_ContinueConversation_InputValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for empty message", func(t *testing.T) {
		// Use nil repository
		mockAssist := &MockAssistant{
			ReplyResponse: "Test Reply",
		}
		srv := chat.NewServer(nil, mockAssist, nil)

		// Try to continue conversation with empty message
		_, err := srv.ContinueConversation(ctx, &pb.ContinueConversationRequest{
			Message: "",
		})

		// Assert error is returned
		if err == nil {
			t.Fatal("expected error for empty message, got nil")
		}

		// Assert it's a required argument error
		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.InvalidArgument {
			t.Errorf("expected twirp.InvalidArgument error, got %v", err)
		}
	})

	t.Run("returns error when no conversation_id or session_metadata provided", func(t *testing.T) {
		// Use nil repository
		mockAssist := &MockAssistant{
			ReplyResponse: "Test Reply",
		}
		srv := chat.NewServer(nil, mockAssist, nil)

		// Try to continue conversation without conversation_id or session_metadata
		_, err := srv.ContinueConversation(ctx, &pb.ContinueConversationRequest{
			Message: "test message",
		})

		// Assert error is returned
		if err == nil {
			t.Fatal("expected error for missing conversation_id or session_metadata, got nil")
		}

		// Assert it's a required argument error
		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.InvalidArgument {
			t.Errorf("expected twirp.InvalidArgument error, got %v", err)
		}
	})
}

func TestServer_DescribeConversation_InputValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for empty conversation_id", func(t *testing.T) {
		// Use nil repository
		srv := chat.NewServer(nil, nil, nil)

		// Try to describe conversation with empty ID
		_, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{
			ConversationId: "",
		})

		// Assert error is returned
		if err == nil {
			t.Fatal("expected error for empty conversation_id, got nil")
		}

		// Assert it's a required argument error
		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.InvalidArgument {
			t.Errorf("expected twirp.InvalidArgument error, got %v", err)
		}
	})
}
