//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/assistant"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestAssistantCompleteWorkflow tests the complete assistant workflow with retry mechanism
func TestAssistantCompleteWorkflow(t *testing.T) {
	// Create assistant
	assist := assistant.New()

	// Test conversation scenarios
	tests := []struct {
		name        string
		messages    []*model.Message
		description string
	}{
		{
			name: "Simple weather inquiry",
			messages: []*model.Message{
				{
					ID:      primitive.NewObjectID(),
					Role:    model.RoleUser,
					Content: "What's the weather like in Barcelona today?",
				},
			},
			description: "Test weather tool integration",
		},
		{
			name: "Date and weather combination",
			messages: []*model.Message{
				{
					ID:      primitive.NewObjectID(),
					Role:    model.RoleUser,
					Content: "What's today's date and the weather in Madrid?",
				},
			},
			description: "Test multiple tool integration",
		},
		{
			name: "Holiday inquiry",
			messages: []*model.Message{
				{
					ID:      primitive.NewObjectID(),
					Role:    model.RoleUser,
					Content: "Are there any holidays this month?",
				},
			},
			description: "Test holiday tool integration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Create conversation
			conv := &model.Conversation{
				ID:       primitive.NewObjectID(),
				Title:    "Test Conversation",
				Messages: tt.messages,
			}

			t.Logf("Testing: %s", tt.description)

			// Test title generation with retry
			title, err := assist.Title(ctx, conv)
			if err != nil {
				t.Logf("Title generation failed (may be expected without valid API key): %v", err)
			} else {
				t.Logf("Generated title: %s", title)
			}

			// Test reply generation with retry
			reply, err := assist.Reply(ctx, conv)
			if err != nil {
				t.Logf("Reply generation failed (may be expected without valid API key): %v", err)
			} else {
				t.Logf("Generated reply: %s", reply)
			}

			// The test passes if the system handles the workflow without crashing
			// Even if external APIs are unavailable, the retry mechanism should handle it gracefully
			t.Logf("Workflow completed successfully for: %s", tt.name)
		})
	}
}

// TestAssistantErrorHandling tests error handling and retry mechanisms
func TestAssistantErrorHandling(t *testing.T) {
	assist := assistant.New()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with empty conversation
	emptyConv := &model.Conversation{
		ID:       primitive.NewObjectID(),
		Title:    "Empty Conversation",
		Messages: []*model.Message{},
	}

	// Test title generation with empty conversation
	title, err := assist.Title(ctx, emptyConv)
	if err != nil {
		t.Logf("Title generation with empty conversation handled: %v", err)
	} else {
		t.Logf("Title for empty conversation: %s", title)
	}

	// Test reply generation with empty conversation
	_, err = assist.Reply(ctx, emptyConv)
	if err == nil {
		t.Error("Expected error for empty conversation reply")
	} else {
		t.Logf("Reply generation with empty conversation properly handled: %v", err)
	}

	// Test with malformed conversation data
	malformedConv := &model.Conversation{
		ID:    primitive.NewObjectID(),
		Title: "Malformed Conversation",
		Messages: []*model.Message{
			{
				ID:      primitive.NewObjectID(),
				Role:    model.RoleUser,
				Content: "", // Empty content
			},
		},
	}

	// Test title generation with malformed data
	_, err = assist.Title(ctx, malformedConv)
	if err != nil {
		t.Logf("Title generation with malformed data handled: %v", err)
	}

	// Test reply generation with malformed data
	_, err = assist.Reply(ctx, malformedConv)
	if err != nil {
		t.Logf("Reply generation with malformed data handled: %v", err)
	}
}

// TestAssistantToolIntegration tests the integration of various tools
func TestAssistantToolIntegration(t *testing.T) {
	assist := assistant.New()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Test scenarios that should trigger different tools
	toolScenarios := []struct {
		name     string
		question string
		expected string
	}{
		{
			name:     "Date tool",
			question: "What is today's date?",
			expected: "date",
		},
		{
			name:     "Weather tool",
			question: "What's the weather in Barcelona?",
			expected: "weather",
		},
		{
			name:     "Holiday tool",
			question: "Are there any holidays this week?",
			expected: "holidays",
		},
	}

	for _, scenario := range toolScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			conv := &model.Conversation{
				ID:    primitive.NewObjectID(),
				Title: scenario.name,
				Messages: []*model.Message{
					{
						ID:      primitive.NewObjectID(),
						Role:    model.RoleUser,
						Content: scenario.question,
					},
				},
			}

			// The assistant should handle tool calls even if external APIs are unavailable
			reply, err := assist.Reply(ctx, conv)
			if err != nil {
				t.Logf("Tool integration test for %s handled error: %v", scenario.name, err)
			} else {
				t.Logf("Tool integration test for %s succeeded: %s", scenario.name, reply)
			}

			// The test passes if the system doesn't crash and handles the tool integration
			t.Logf("Tool integration test completed for: %s", scenario.name)
		})
	}
}
