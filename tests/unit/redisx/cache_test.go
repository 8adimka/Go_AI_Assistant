package package package redisx_test

import (
	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewCache(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ttl := 1 * time.Hour
	cache := NewCache(client, ttl)

	if cache == nil {
		t.Fatal("Expected cache to be non-nil")
	}

	if cache.client != client {
		t.Error("Expected client to be set correctly")
	}

	if cache.ttl != ttl {
		t.Errorf("Expected TTL to be %v, got %v", ttl, cache.ttl)
	}
}

func TestCache_GenerateKey(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	cache := NewCache(client, 1*time.Hour)

	tests := []struct {
		name     string
		prefix   string
		content  string
		wantLen  int // prefix + ":" + 64 hex chars
	}{
		{
			name:    "simple content",
			prefix:  "test",
			content: "hello world",
			wantLen: len("test:") + 64, // SHA256 produces 64 hex characters
		},
		{
			name:    "sensitive content",
			prefix:  "title",
			content: "What is my credit card number 1234-5678-9012-3456?",
			wantLen: len("title:") + 64,
		},
		{
			name:    "long content",
			prefix:  "weather",
			content: "This is a very long message that could contain sensitive information and should be hashed to prevent it from appearing in Redis keys where it might be logged or exposed",
			wantLen: len("weather:") + 64,
		},
		{
			name:    "empty content",
			prefix:  "empty",
			content: "",
			wantLen: len("empty:") + 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cache.GenerateKey(tt.prefix, tt.content)

			// Check key length
			if len(key) != tt.wantLen {
				t.Errorf("Expected key length %d, got %d", tt.wantLen, len(key))
			}

			// Check key format: prefix:hash
			expectedPrefix := tt.prefix + ":"
			if len(key) < len(expectedPrefix) {
				t.Fatalf("Key too short: %s", key)
			}

			if key[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("Expected key to start with %q, got %q", expectedPrefix, key[:len(expectedPrefix)])
			}

			// Verify hash part is valid hex
			hashPart := key[len(expectedPrefix):]
			if len(hashPart) != 64 {
				t.Errorf("Expected hash length 64, got %d", len(hashPart))
			}

			// Verify it's valid hex
			if _, err := hex.DecodeString(hashPart); err != nil {
				t.Errorf("Hash part is not valid hex: %v", err)
			}

			// Verify the hash matches expected SHA256
			expectedHash := sha256.Sum256([]byte(tt.content))
			expectedKey := tt.prefix + ":" + hex.EncodeToString(expectedHash[:])
			if key != expectedKey {
				t.Errorf("Expected key %q, got %q", expectedKey, key)
			}
		})
	}
}

func TestCache_GenerateKey_Deterministic(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	cache := NewCache(client, 1*time.Hour)

	content := "test content"
	prefix := "test"

	// Generate key multiple times
	key1 := cache.GenerateKey(prefix, content)
	key2 := cache.GenerateKey(prefix, content)
	key3 := cache.GenerateKey(prefix, content)

	// All should be identical
	if key1 != key2 || key2 != key3 {
		t.Errorf("Keys should be deterministic. Got: %q, %q, %q", key1, key2, key3)
	}
}

func TestCache_GenerateKey_DifferentContent(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	cache := NewCache(client, 1*time.Hour)

	prefix := "test"
	content1 := "content one"
	content2 := "content two"

	key1 := cache.GenerateKey(prefix, content1)
	key2 := cache.GenerateKey(prefix, content2)

	// Keys should be different
	if key1 == key2 {
		t.Errorf("Different content should produce different keys. Got same key: %q", key1)
	}

	// Both should have the same prefix
	expectedPrefix := prefix + ":"
	if key1[:len(expectedPrefix)] != expectedPrefix || key2[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Both keys should have prefix %q", expectedPrefix)
	}
}

func TestCache_GenerateKey_NoSensitiveDataInKey(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	cache := NewCache(client, 1*time.Hour)

	sensitiveContent := "password123 credit_card:1234-5678-9012-3456 ssn:123-45-6789"
	key := cache.GenerateKey("sensitive", sensitiveContent)

	// Verify the sensitive content is NOT in the key
	if contains(key, "password") {
		t.Error("Key should not contain 'password'")
	}
	if contains(key, "credit_card") {
		t.Error("Key should not contain 'credit_card'")
	}
	if contains(key, "1234") {
		t.Error("Key should not contain credit card digits")
	}
	if contains(key, "ssn") {
		t.Error("Key should not contain 'ssn'")
	}
	if contains(key, "123-45") {
		t.Error("Key should not contain SSN")
	}

	// Key should only contain prefix and hex hash
	t.Logf("Generated safe key: %s", key)
}

func TestCache_TTL_Is_Set_Correctly(t *testing.T) {
	testCases := []struct {
		name string
		ttl  time.Duration
	}{
		{"1 hour", 1 * time.Hour},
		{"24 hours", 24 * time.Hour},
		{"30 minutes", 30 * time.Minute},
		{"1 week", 7 * 24 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{
				Addr: "localhost:6379",
			})
			cache := NewCache(client, tc.ttl)

			if cache.ttl != tc.ttl {
				t.Errorf("Expected TTL %v, got %v", tc.ttl, cache.ttl)
			}
		})
	}
}

func TestCache_SetAndGet(t *testing.T) {
	// This test requires a running Redis instance
	// Skip if Redis is not available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	cache := NewCache(client, 1*time.Minute)

	// Test data
	type TestData struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	testData := TestData{
		Message: "test message",
		Count:   42,
	}

	// Generate a safe key
	key := cache.GenerateKey("test", "test-content")

	// Set value
	err := cache.Set(ctx, key, testData)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Get value
	var retrieved TestData
	err = cache.Get(ctx, key, &retrieved)
	if err != nil {
		t.Fatalf("Failed to get from cache: %v", err)
	}

	// Verify
	if retrieved.Message != testData.Message {
		t.Errorf("Expected message %q, got %q", testData.Message, retrieved.Message)
	}
	if retrieved.Count != testData.Count {
		t.Errorf("Expected count %d, got %d", testData.Count, retrieved.Count)
	}

	// Cleanup
	cache.Delete(ctx, key)
}

func TestCache_GetMiss(t *testing.T) {
	// This test requires a running Redis instance
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	cache := NewCache(client, 1*time.Minute)
	key := cache.GenerateKey("test", "nonexistent-key")

	var data string
	err := cache.Get(ctx, key, &data)

	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
