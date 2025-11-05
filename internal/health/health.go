package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// HealthChecker handles health checks
type HealthChecker struct {
	mongoClient *mongo.Client
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(mongoClient *mongo.Client) *HealthChecker {
	return &HealthChecker{
		mongoClient: mongoClient,
	}
}

// HealthHandler handles the /health endpoint
func (h *HealthChecker) HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	// Check MongoDB connection
	if h.mongoClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := h.mongoClient.Ping(ctx, nil); err != nil {
			response.Status = "unhealthy"
			response.Checks["mongodb"] = "failed: " + err.Error()
		} else {
			response.Checks["mongodb"] = "ok"
		}
	} else {
		response.Checks["mongodb"] = "not configured"
	}

	// Set response status code
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// ReadyHandler handles the /ready endpoint
func (h *HealthChecker) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ready",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	// Check MongoDB connection for readiness
	if h.mongoClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := h.mongoClient.Ping(ctx, nil); err != nil {
			response.Status = "not ready"
			response.Checks["mongodb"] = "failed: " + err.Error()
		} else {
			response.Checks["mongodb"] = "ok"
		}
	} else {
		response.Status = "not ready"
		response.Checks["mongodb"] = "not configured"
	}

	// Set response status code
	statusCode := http.StatusOK
	if response.Status == "not ready" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
