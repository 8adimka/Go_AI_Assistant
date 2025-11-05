package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	. "github.com/8adimka/Go_AI_Assistant/internal/chat/testing"
	"github.com/8adimka/Go_AI_Assistant/internal/pb"
	"github.com/google/go-cmp/cmp"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/testing/protocmp"
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

func TestServer_DescribeConversation(t *testing.T) {
	ctx := context.Background()
	srv := NewServer(model.New(ConnectMongo()), nil)

	t.Run("describe existing conversation", WithFixture(func(t *testing.T, f *Fixture) {
		c := f.CreateConversation()

		out, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: c.ID.Hex()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, want := out.GetConversation(), c.Proto()
		if !cmp.Equal(got, want, protocmp.Transform()) {
			t.Errorf("DescribeConversation() mismatch (-got +want):\n%s", cmp.Diff(got, want, protocmp.Transform()))
		}
	}))

	t.Run("describe non existing conversation should return 404", WithFixture(func(t *testing.T, f *Fixture) {
		_, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: "08a59244257c872c5943e2a2"})
		if err == nil {
			t.Fatal("expected error for non-existing conversation, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.NotFound {
			t.Fatalf("expected twirp.NotFound error, got %v", err)
		}
	}))
}

func TestServer_StartConversation(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully starts conversation with valid message", WithFixture(func(t *testing.T, f *Fixture) {
		// Create mock assistant with deterministic responses
		mockAssist := &MockAssistant{
			TitleResponse: "Weather in Barcelona",
			ReplyResponse: "Sunny 20C",
		}
		srv := NewServer(f.Repository, mockAssist)

		// Start conversation
		resp, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "What is the weather like in Barcelona?",
		})

		// Assert no error
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert response has conversation ID
		if resp.ConversationId == "" {
			t.Error("expected non-empty conversation_id")
		}

		// Assert title matches expected
		if resp.Title != "Weather in Barcelona" {
			t.Errorf("expected title 'Weather in Barcelona', got %q", resp.Title)
		}

		// Assert reply matches expected
		if resp.Reply != "Sunny 20C" {
			t.Errorf("expected reply 'Sunny 20C', got %q", resp.Reply)
		}

		// Verify conversation was stored in database
		conv, err := f.Repository.DescribeConversation(ctx, resp.ConversationId)
		if err != nil {
			t.Fatalf("failed to retrieve stored conversation: %v", err)
		}

		// Verify conversation has correct title
		if conv.Title != "Weather in Barcelona" {
			t.Errorf("stored conversation title = %q, want %q", conv.Title, "Weather in Barcelona")
		}

		// Verify conversation has 2 messages (user + assistant)
		if len(conv.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(conv.Messages))
		}

		// Verify first message is user message
		if conv.Messages[0].Role != model.RoleUser {
			t.Errorf("first message role = %v, want %v", conv.Messages[0].Role, model.RoleUser)
		}
		if conv.Messages[0].Content != "What is the weather like in Barcelona?" {
			t.Errorf("first message content = %q, want %q", conv.Messages[0].Content, "What is the weather like in Barcelona?")
		}

		// Verify second message is assistant message
		if conv.Messages[1].Role != model.RoleAssistant {
			t.Errorf("second message role = %v, want %v", conv.Messages[1].Role, model.RoleAssistant)
		}
		if conv.Messages[1].Content != "Sunny 20C" {
			t.Errorf("second message content = %q, want %q", conv.Messages[1].Content, "Sunny 20C")
		}
	}))

	t.Run("returns error for empty message", WithFixture(func(t *testing.T, f *Fixture) {
		mockAssist := &MockAssistant{
			TitleResponse: "Test Title",
			ReplyResponse: "Test Reply",
		}
		srv := NewServer(f.Repository, mockAssist)

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
	}))

	t.Run("handles title generation error gracefully", WithFixture(func(t *testing.T, f *Fixture) {
		// Create mock assistant that fails on title generation
		mockAssist := &MockAssistant{
			TitleError:    twirp.InternalError("title generation failed"),
			ReplyResponse: "Sunny 20C",
		}
		srv := NewServer(f.Repository, mockAssist)

		// Start conversation should still succeed even if title fails
		resp, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "What is the weather like in Barcelona?",
		})

		// Assert no error (title generation errors are logged but not fatal)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert fallback title is used
		if resp.Title != "Untitled conversation" {
			t.Errorf("expected fallback title 'Untitled conversation', got %q", resp.Title)
		}

		// Assert reply is still generated
		if resp.Reply != "Sunny 20C" {
			t.Errorf("expected reply 'Sunny 20C', got %q", resp.Reply)
		}
	}))

	t.Run("returns error when reply generation fails", WithFixture(func(t *testing.T, f *Fixture) {
		// Create mock assistant that fails on reply generation
		mockAssist := &MockAssistant{
			TitleResponse: "Weather in Barcelona",
			ReplyError:    twirp.InternalError("reply generation failed"),
		}
		srv := NewServer(f.Repository, mockAssist)

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
	}))

	t.Run("handles whitespace-only message as empty", WithFixture(func(t *testing.T, f *Fixture) {
		mockAssist := &MockAssistant{
			TitleResponse: "Test Title",
			ReplyResponse: "Test Reply",
		}
		srv := NewServer(f.Repository, mockAssist)

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
	}))
}
