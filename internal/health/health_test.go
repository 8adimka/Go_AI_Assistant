package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestHealthHandler_BothServicesHealthy(t *testing.T) {
	// Connect to MongoDB for testing
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping integration test")
	}
	defer mongoClient.Disconnect(context.Background())

	// Connect to Redis for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.Close()

	checker := NewHealthChecker(mongoClient, redisClient)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	checker.HealthHandler(rec, req)

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Parse response
	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check overall status
	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got %q", response.Status)
	}

	// Check MongoDB status
	if response.Checks["mongodb"] != "ok" {
		t.Errorf("Expected MongoDB status 'ok', got %q", response.Checks["mongodb"])
	}

	// Check Redis status
	if response.Checks["redis"] != "ok" {
		t.Errorf("Expected Redis status 'ok', got %q", response.Checks["redis"])
	}

	// Check timestamp is recent
	if time.Since(response.Timestamp) > 5*time.Second {
		t.Error("Timestamp is too old")
	}
}

func TestHealthHandler_NoMongoDB(t *testing.T) {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.Close()

	checker := NewHealthChecker(nil, redisClient)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	checker.HealthHandler(rec, req)

	// Parse response
	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check MongoDB status
	if response.Checks["mongodb"] != "not configured" {
		t.Errorf("Expected MongoDB status 'not configured', got %q", response.Checks["mongodb"])
	}

	// Redis should still be ok
	if response.Checks["redis"] != "ok" {
		t.Errorf("Expected Redis status 'ok', got %q", response.Checks["redis"])
	}
}

func TestHealthHandler_NoRedis(t *testing.T) {
	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping integration test")
	}
	defer mongoClient.Disconnect(context.Background())

	checker := NewHealthChecker(mongoClient, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	checker.HealthHandler(rec, req)

	// Parse response
	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// MongoDB should be ok
	if response.Checks["mongodb"] != "ok" {
		t.Errorf("Expected MongoDB status 'ok', got %q", response.Checks["mongodb"])
	}

	// Check Redis status
	if response.Checks["redis"] != "not configured" {
		t.Errorf("Expected Redis status 'not configured', got %q", response.Checks["redis"])
	}
}

func TestHealthHandler_RedisDown(t *testing.T) {
	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping integration test")
	}
	defer mongoClient.Disconnect(context.Background())

	// Create Redis client with invalid address
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Invalid port
	})
	defer redisClient.Close()

	checker := NewHealthChecker(mongoClient, redisClient)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	checker.HealthHandler(rec, req)

	// Check status code should be 503 Service Unavailable
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	// Parse response
	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check overall status
	if response.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got %q", response.Status)
	}

	// MongoDB should still be ok
	if response.Checks["mongodb"] != "ok" {
		t.Errorf("Expected MongoDB status 'ok', got %q", response.Checks["mongodb"])
	}

	// Redis should be failed
	if response.Checks["redis"] == "ok" {
		t.Error("Expected Redis status to indicate failure")
	}
	if response.Checks["redis"] != "" && response.Checks["redis"][:7] != "failed:" {
		t.Errorf("Expected Redis status to start with 'failed:', got %q", response.Checks["redis"])
	}
}

func TestReadyHandler_BothServicesReady(t *testing.T) {
	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping integration test")
	}
	defer mongoClient.Disconnect(context.Background())

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.Close()

	checker := NewHealthChecker(mongoClient, redisClient)

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()

	checker.ReadyHandler(rec, req)

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Parse response
	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check overall status
	if response.Status != "ready" {
		t.Errorf("Expected status 'ready', got %q", response.Status)
	}

	// Check MongoDB status
	if response.Checks["mongodb"] != "ok" {
		t.Errorf("Expected MongoDB status 'ok', got %q", response.Checks["mongodb"])
	}

	// Check Redis status
	if response.Checks["redis"] != "ok" {
		t.Errorf("Expected Redis status 'ok', got %q", response.Checks["redis"])
	}
}

func TestReadyHandler_RedisNotReady(t *testing.T) {
	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping integration test")
	}
	defer mongoClient.Disconnect(context.Background())

	// Redis not configured
	checker := NewHealthChecker(mongoClient, nil)

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()

	checker.ReadyHandler(rec, req)

	// Check status code should be 503
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	// Parse response
	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check overall status
	if response.Status != "not ready" {
		t.Errorf("Expected status 'not ready', got %q", response.Status)
	}

	// Redis should not be configured
	if response.Checks["redis"] != "not configured" {
		t.Errorf("Expected Redis status 'not configured', got %q", response.Checks["redis"])
	}
}

func TestHealthResponse_JSON(t *testing.T) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
		Checks: map[string]string{
			"mongodb": "ok",
			"redis":   "ok",
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var decoded HealthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decoded.Status != response.Status {
		t.Errorf("Status mismatch: got %q, want %q", decoded.Status, response.Status)
	}

	if decoded.Checks["mongodb"] != "ok" {
		t.Errorf("MongoDB check mismatch: got %q, want 'ok'", decoded.Checks["mongodb"])
	}

	if decoded.Checks["redis"] != "ok" {
		t.Errorf("Redis check mismatch: got %q, want 'ok'", decoded.Checks["redis"])
	}
}
