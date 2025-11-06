//go:build integration

package redisx_test

import (
	"context"
	"testing"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/redisx"
	"github.com/redis/go-redis/v9"
)

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

	cache := redisx.NewCache(client, 1*time.Minute)

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

	cache := redisx.NewCache(client, 1*time.Minute)
	key := cache.GenerateKey("test", "nonexistent-key")

	var data string
	err := cache.Get(ctx, key, &data)

	if err != redisx.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}
