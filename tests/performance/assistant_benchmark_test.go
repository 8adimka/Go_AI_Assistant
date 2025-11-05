package performance

import (
	"context"
	"testing"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/assistant"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/tests/unit/mocks"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BenchmarkAssistant_Reply benchmarks the Reply method performance
func BenchmarkAssistant_Reply(b *testing.B) {
	// Create mock OpenAI client
	mockClient := mocks.NewMockOpenAIClient()
	
	// Create assistant with mock client
	assist := &assistant.Assistant{
		// We need to set the internal fields, but they're private
		// This benchmark will need to be updated when we can properly inject dependencies
	}

	// Create test conversation
	conv := &model.Conversation{
		ID:    primitive.NewObjectID(),
		Title: "Test Conversation",
		Messages: []*model.Message{
			{
				ID:      primitive.NewObjectID(),
				Role:    model.RoleUser,
				Content: "What is the weather like today?",
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := assist.Reply(ctx, conv)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkAssistant_Title benchmarks the Title method performance
func BenchmarkAssistant_Title(b *testing.B) {
	// Create assistant
	assist := &assistant.Assistant{}

	// Create test conversation
	conv := &model.Conversation{
		ID:    primitive.NewObjectID(),
		Title: "Test Conversation",
		Messages: []*model.Message{
			{
				ID:      primitive.NewObjectID(),
				Role:    model.RoleUser,
				Content: "What is the weather like today?",
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := assist.Title(ctx, conv)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkAssistant_formatTitle benchmarks the title formatting performance
func BenchmarkAssistant_formatTitle(b *testing.B) {
	assist := &assistant.Assistant{}

	testCases := []struct {
		name  string
		input string
	}{
		{"Short title", "weather in barcelona"},
		{"Long title", "This is a very long title that should be truncated because it exceeds the character limit"},
		{"With special chars", "\"Weather in Barcelona\" - with quotes"},
		{"Mixed case", "mAcHiNe LeArNiNg AlGoRiThMs"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = assist.formatTitle(tc.input)
			}
		})
	}
}

// BenchmarkAssistant_toTitleCase benchmarks the title case conversion performance
func BenchmarkAssistant_toTitleCase(b *testing.B) {
	assist := &assistant.Assistant{}

	testCases := []struct {
		name  string
		input string
	}{
		{"Simple case", "hello world"},
		{"With short words", "the cat in the hat"},
		{"Mixed case", "mAcHiNe LeArNiNg"},
		{"Single word", "weather"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = assist.toTitleCase(tc.input)
			}
		})
	}
}
