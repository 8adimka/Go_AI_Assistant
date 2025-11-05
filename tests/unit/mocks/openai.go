package mocks

import (
	"context"

	"github.com/openai/openai-go/v2"
)

// MockOpenAIClient is a mock implementation of openai.Client for testing
type MockOpenAIClient struct {
	// Mock responses
	ChatCompletionResponse *openai.ChatCompletion
	ChatCompletionError    error

	// Call tracking
	ChatCompletionCallCount int
	LastChatCompletionParams *openai.ChatCompletionNewParams
}

// NewMockOpenAIClient creates a new mock OpenAI client
func NewMockOpenAIClient() *MockOpenAIClient {
	return &MockOpenAIClient{
		ChatCompletionResponse: &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: openai.String("Mock response from AI assistant"),
					},
				},
			},
		},
	}
}

// Chat provides access to chat completion methods
func (m *MockOpenAIClient) Chat() *openai.ChatService {
	return &openai.ChatService{
		Completions: openai.ChatCompletionService{},
	}
}

// New creates a new chat completion
func (m *MockOpenAIClient) New(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	m.ChatCompletionCallCount++
	m.LastChatCompletionParams = &params

	if m.ChatCompletionError != nil {
		return nil, m.ChatCompletionError
	}

	return m.ChatCompletionResponse, nil
}

// WithChatCompletionResponse sets the mock response for chat completions
func (m *MockOpenAIClient) WithChatCompletionResponse(response *openai.ChatCompletion) *MockOpenAIClient {
	m.ChatCompletionResponse = response
	return m
}

// WithChatCompletionError sets the mock error for chat completions
func (m *MockOpenAIClient) WithChatCompletionError(err error) *MockOpenAIClient {
	m.ChatCompletionError = err
	return m
}

// ResetCallCount resets the call counters
func (m *MockOpenAIClient) ResetCallCount() {
	m.ChatCompletionCallCount = 0
	m.LastChatCompletionParams = nil
}

// MockChatCompletion creates a mock chat completion response
func MockChatCompletion(content string) *openai.ChatCompletion {
	return &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: openai.F(content),
				},
			},
		},
	}
}
