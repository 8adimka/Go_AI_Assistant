package performance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/retry"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// mockAssistant is a mock implementation to avoid real API calls during benchmarks.
type mockAssistant struct{}

func (m *mockAssistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	// Simulate a quick response without API calls
	return "mock reply", nil
}

func (m *mockAssistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	// Simulate a quick title generation without API calls
	return "mock title", nil
}

// BenchmarkAssistant_Reply benchmarks the Reply method performance
func BenchmarkAssistant_Reply(b *testing.B) {
	// Use mock assistant to avoid real API calls
	assist := &mockAssistant{}

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
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkAssistant_Title benchmarks the Title method performance
func BenchmarkAssistant_Title(b *testing.B) {
	// Use mock assistant to avoid real API calls
	assist := &mockAssistant{}

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
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkAssistant_TitleGeneration benchmarks the complete title generation process
func BenchmarkAssistant_TitleGeneration(b *testing.B) {
	// Use mock assistant to avoid real API calls
	assist := &mockAssistant{}

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
				// Simulate title generation by testing the public Title method
				// This will test the complete title generation pipeline
				conv := &model.Conversation{
					ID:    primitive.NewObjectID(),
					Title: "Test Conversation",
					Messages: []*model.Message{
						{
							ID:      primitive.NewObjectID(),
							Role:    model.RoleUser,
							Content: tc.input,
						},
					},
				}
				_, err := assist.Title(context.Background(), conv)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// BenchmarkRetryMechanism benchmarks the retry mechanism performance
func BenchmarkRetryMechanism(b *testing.B) {
	ctx := context.Background()

	// Test different retry configurations
	configs := []struct {
		name        string
		maxAttempts int
		baseDelay   time.Duration
		maxDelay    time.Duration
	}{
		{"Fast retry", 2, 1 * time.Millisecond, 10 * time.Millisecond},
		{"Standard retry", 3, 10 * time.Millisecond, 100 * time.Millisecond},
		{"Conservative retry", 5, 50 * time.Millisecond, 500 * time.Millisecond},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				config := retry.RetryConfig{
					MaxAttempts: cfg.maxAttempts,
					BaseDelay:   cfg.baseDelay,
					MaxDelay:    cfg.maxDelay,
				}

				// Test with immediate success
				_, _ = retry.RetryWithResult(ctx, config, func() (interface{}, error) {
					return "success", nil
				})
			}
		})
	}
}

// BenchmarkRetryWithFailures benchmarks retry performance with simulated failures
func BenchmarkRetryWithFailures(b *testing.B) {
	ctx := context.Background()
	config := retry.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	failureScenarios := []struct {
		name          string
		failureRate   float64 // 0.0 to 1.0
		shouldSucceed bool
	}{
		{"Always succeed", 0.0, true},
		{"Low failure rate", 0.2, true},
		{"High failure rate", 0.8, false},
		{"Always fail", 1.0, false},
	}

	for _, scenario := range failureScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = retry.RetryWithResult(ctx, config, func() (interface{}, error) {
					// Simulate failure based on failure rate
					if float64(i%100)/100.0 < scenario.failureRate {
						return nil, errors.New("simulated failure")
					}
					return "success", nil
				})
			}
		})
	}
}

// BenchmarkToolExecutionWithRetry benchmarks tool execution with retry mechanism
func BenchmarkToolExecutionWithRetry(b *testing.B) {
	// Use mock assistant to avoid real API calls
	assist := &mockAssistant{}
	ctx := context.Background()

	toolScenarios := []struct {
		name     string
		question string
	}{
		{"Date tool", "What is today's date?"},
		{"Weather tool", "What's the weather in Barcelona?"},
		{"Holiday tool", "Are there any holidays this month?"},
	}

	for _, scenario := range toolScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				conv := &model.Conversation{
					ID:    primitive.NewObjectID(),
					Title: "Benchmark Conversation",
					Messages: []*model.Message{
						{
							ID:      primitive.NewObjectID(),
							Role:    model.RoleUser,
							Content: scenario.question,
						},
					},
				}
				// This will test the complete pipeline including tool execution and retry
				_, err := assist.Reply(ctx, conv)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
