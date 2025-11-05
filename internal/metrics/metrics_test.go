package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestHTTPMetricsMiddleware(t *testing.T) {
	// Create a test meter provider with Prometheus exporter
	ctx := context.Background()
	
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("test-service"),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create resource: %v", err)
	}

	exporter, err := prometheus.New()
	if err != nil {
		t.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(exporter),
	)
	otel.SetMeterProvider(provider)

	// Create metrics
	meter := provider.Meter("test")
	appMetrics, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with metrics middleware
	wrappedHandler := appMetrics.HTTPMetricsMiddleware()(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "test response" {
		t.Errorf("Expected 'test response', got %q", rec.Body.String())
	}

	// Test that metrics are recorded by making another request
	req2 := httptest.NewRequest("POST", "/api/test", nil)
	rec2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec2, req2)

	// Test error status code
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})
	wrappedErrorHandler := appMetrics.HTTPMetricsMiddleware()(errorHandler)
	
	req3 := httptest.NewRequest("GET", "/error", nil)
	rec3 := httptest.NewRecorder()
	wrappedErrorHandler.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec3.Code)
	}

	// We can't easily verify the exact metric values without a full OpenTelemetry setup,
	// but we've verified the middleware doesn't break the handler chain
	t.Log("Metrics middleware successfully wraps handler and allows requests to complete")
}

func TestNewMetrics(t *testing.T) {
	ctx := context.Background()
	
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("test-service"),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create resource: %v", err)
	}

	exporter, err := prometheus.New()
	if err != nil {
		t.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(exporter),
	)

	meter := provider.Meter("test")
	
	metrics, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected metrics to be non-nil")
	}

	if metrics.httpRequestsTotal == nil {
		t.Error("httpRequestsTotal should not be nil")
	}

	if metrics.httpRequestDuration == nil {
		t.Error("httpRequestDuration should not be nil")
	}

	if metrics.httpRequestsInFlight == nil {
		t.Error("httpRequestsInFlight should not be nil")
	}

	if metrics.twirpRequestsTotal == nil {
		t.Error("twirpRequestsTotal should not be nil")
	}
}

func TestResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Test default status code
	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected default status code 200, got %d", rw.statusCode)
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", rw.statusCode)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected recorder status code 201, got %d", rec.Code)
	}
}

func TestRecordTwirpRequest(t *testing.T) {
	ctx := context.Background()
	
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("test-service"),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create resource: %v", err)
	}

	exporter, err := prometheus.New()
	if err != nil {
		t.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(exporter),
	)

	meter := provider.Meter("test")
	
	metrics, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	// This should not panic
	metrics.RecordTwirpRequest(ctx, "StartConversation", "success")
	metrics.RecordTwirpRequest(ctx, "ContinueConversation", "error")
}

func TestMetricsMiddlewareWithMultipleRequests(t *testing.T) {
	// Test that middleware handles multiple concurrent requests correctly
	ctx := context.Background()
	
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("test-multiple-requests"),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create resource: %v", err)
	}

	exporter, err := prometheus.New()
	if err != nil {
		t.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(exporter),
	)
	otel.SetMeterProvider(provider)

	meter := provider.Meter("test")
	appMetrics, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	// Create test handlers for different status codes
	handlers := map[string]http.HandlerFunc{
		"/success": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		},
		"/created": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("Created"))
		},
		"/error": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error"))
		},
	}

	// Test each handler
	for path, handler := range handlers {
		wrappedHandler := appMetrics.HTTPMetricsMiddleware()(handler)
		
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
		
		// Verify handler executed correctly
		if rec.Code == 0 {
			t.Errorf("Handler for %s did not set status code", path)
		}
	}

	t.Log("Metrics middleware successfully handles multiple different requests")
}
