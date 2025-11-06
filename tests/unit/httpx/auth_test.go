package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/8adimka/Go_AI_Assistant/internal/httpx"
)

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	auth := httpx.NewAPIKeyAuth("secret-key-123")

	handler := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "secret-key-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "success" {
		t.Errorf("Expected body 'success', got %q", rec.Body.String())
	}
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	auth := httpx.NewAPIKeyAuth("secret-key-123")

	handler := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	// No X-API-Key header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	if rec.Header().Get("WWW-Authenticate") != "API-Key" {
		t.Error("Expected WWW-Authenticate header")
	}

	body := rec.Body.String()
	if body == "" {
		t.Error("Expected error message in body")
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	auth := httpx.NewAPIKeyAuth("secret-key-123")

	handler := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_NoKeyConfigured(t *testing.T) {
	auth := httpx.NewAPIKeyAuth("") // No API key configured

	handler := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	// No X-API-Key header provided
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should allow through when no API key is configured
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 when no API key configured, got %d", rec.Code)
	}
}

func TestConstantTimeCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{"identical strings", "secret", "secret", true},
		{"different strings same length", "secret", "public", false},
		{"different lengths", "secret", "secrets", false},
		{"empty strings", "", "", true},
		{"one empty", "secret", "", false},
		{"case sensitive", "Secret", "secret", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := httpx.ConstantTimeCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("constantTimeCompare(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestProtectedRoutes_PublicPath(t *testing.T) {
	publicPaths := []string{"/health", "/ready", "/twirp/*"}
	middleware := httpx.ProtectedRoutes("secret-key", publicPaths)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test public paths (should not require API key)
	publicTests := []string{
		"/health",
		"/ready",
		"/twirp/chat.ChatService/SendMessage",
		"/twirp/chat.ChatService/StartConversation",
	}

	for _, path := range publicTests {
		req := httptest.NewRequest("GET", path, nil)
		// No API key provided
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Public path %s: expected status 200, got %d", path, rec.Code)
		}
	}
}

func TestProtectedRoutes_ProtectedPath(t *testing.T) {
	publicPaths := []string{"/health", "/ready", "/twirp/*"}
	middleware := httpx.ProtectedRoutes("secret-key", publicPaths)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test protected paths (should require API key)
	protectedPaths := []string{
		"/metrics",
		"/admin",
		"/admin/users",
		"/api/internal",
	}

	for _, path := range protectedPaths {
		// Without API key - should fail
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Protected path %s without key: expected status 401, got %d", path, rec.Code)
		}

		// With valid API key - should succeed
		req = httptest.NewRequest("GET", path, nil)
		req.Header.Set("X-API-Key", "secret-key")
		rec = httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Protected path %s with key: expected status 200, got %d", path, rec.Code)
		}
	}
}

func TestMatchesPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{"exact match", "/health", "/health", true},
		{"no match", "/health", "/metrics", false},
		{"wildcard match", "/api/users", "/api/*", true},
		{"wildcard match nested", "/api/users/123", "/api/*", true},
		{"wildcard no match", "/admin/users", "/api/*", false},
		{"wildcard base path", "/api", "/api/*", true},
		{"empty path", "", "", true},
		{"trailing slash", "/api/", "/api/*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := httpx.MatchesPath(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPath(%q, %q) = %v, want %v", tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestAPIKeyAuth_MultipleRequests(t *testing.T) {
	auth := httpx.NewAPIKeyAuth("secret-key-123")

	handler := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Multiple requests with valid key
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "secret-key-123")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i, rec.Code)
		}
	}

	// Request with invalid key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Invalid key: expected status 401, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_CaseSensitive(t *testing.T) {
	auth := httpx.NewAPIKeyAuth("Secret-Key-123")

	handler := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Correct case
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "Secret-Key-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Error("Correct case should be accepted")
	}

	// Wrong case
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "secret-key-123")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Error("Wrong case should be rejected")
	}
}
