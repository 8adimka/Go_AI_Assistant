package httpx

import (
	"log/slog"
	"net/http"
	"strings"
)

// APIKeyAuth provides API key authentication middleware
type APIKeyAuth struct {
	apiKey string
}

// NewAPIKeyAuth creates a new API key authentication middleware
func NewAPIKeyAuth(apiKey string) *APIKeyAuth {
	return &APIKeyAuth{
		apiKey: apiKey,
	}
}

// Middleware returns an HTTP middleware that enforces API key authentication
// Checks X-API-Key header against configured API key
func (a *APIKeyAuth) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if API key is not configured (optional auth)
			if a.apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Get API key from header
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				slog.WarnContext(r.Context(), "API key missing",
					"ip", getClientIP(r),
					"method", r.Method,
					"path", r.URL.Path,
				)
				a.unauthorized(w, "API key required")
				return
			}

			// Constant-time comparison to prevent timing attacks
			if !constantTimeCompare(providedKey, a.apiKey) {
				slog.WarnContext(r.Context(), "Invalid API key",
					"ip", getClientIP(r),
					"method", r.Method,
					"path", r.URL.Path,
				)
				a.unauthorized(w, "Invalid API key")
				return
			}

			// API key is valid
			next.ServeHTTP(w, r)
		})
	}
}

// unauthorized sends a 401 Unauthorized response
func (a *APIKeyAuth) unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "API-Key")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized","message":"` + message + `"}`))
}

// constantTimeCompare performs constant-time string comparison
// This prevents timing attacks that could reveal the API key
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// ProtectedRoutes returns a middleware that protects specific routes
// Public routes are allowed without authentication
func ProtectedRoutes(apiKey string, publicPaths []string) func(http.Handler) http.Handler {
	auth := NewAPIKeyAuth(apiKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is public
			for _, publicPath := range publicPaths {
				if matchesPath(r.URL.Path, publicPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Path is protected, require API key
			auth.Middleware()(next).ServeHTTP(w, r)
		})
	}
}

// matchesPath checks if a request path matches a pattern
// Supports exact matches and wildcard patterns (e.g., /api/*)
func matchesPath(path, pattern string) bool {
	// Exact match
	if path == pattern {
		return true
	}

	// Wildcard match (e.g., /api/*)
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}

	return false
}
