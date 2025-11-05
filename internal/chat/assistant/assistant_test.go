package assistant

import (
	"context"
	"testing"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAssistant_formatTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic formatting",
			input:    "  weather in barcelona  ",
			expected: "Weather in Barcelona",
		},
		{
			name:     "with newlines",
			input:    "weather\nin\nbarcelona",
			expected: "Weather in Barcelona",
		},
		{
			name:     "with quotes",
			input:    "\"Weather in Barcelona\"",
			expected: "Weather in Barcelona",
		},
		{
			name:     "too long",
			input:    "This is a very long title that should be truncated because it exceeds the character limit",
			expected: "This Is a Very Long Title That Should Be Truncated Because I",
		},
		{
			name:     "title case conversion",
			input:    "machine learning algorithms overview",
			expected: "Machine Learning Algorithms Overview",
		},
		{
			name:     "short words lowercase",
			input:    "the art of war",
			expected: "The Art of War",
		},
	}

	assistant := &Assistant{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assistant.formatTitle(tt.input)
			if result != tt.expected {
				t.Errorf("formatTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAssistant_toTitleCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple case",
			input:    "hello world",
			expected: "Hello World",
		},
		{
			name:     "with short words",
			input:    "the cat in the hat",
			expected: "The Cat in the Hat",
		},
		{
			name:     "mixed case",
			input:    "mAcHiNe LeArNiNg",
			expected: "Machine Learning",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single word",
			input:    "weather",
			expected: "Weather",
		},
	}

	assistant := &Assistant{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assistant.toTitleCase(tt.input)
			if result != tt.expected {
				t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAssistant_Title_EmptyConversation(t *testing.T) {
	assistant := &Assistant{}

	conv := &model.Conversation{
		ID:       primitive.NewObjectID(),
		Title:    "Untitled conversation",
		Messages: []*model.Message{},
	}

	title, err := assistant.Title(context.Background(), conv)
	if err != nil {
		t.Errorf("Unexpected error for empty conversation: %v", err)
	}

	expected := "An empty conversation"
	if title != expected {
		t.Errorf("Expected %q for empty conversation, got %q", expected, title)
	}
}
