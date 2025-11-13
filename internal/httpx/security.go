package httpx

import (
	"log/slog"
	"net/http"
)

// SecurityMiddleware provides consolidated security middleware functionality
// Combines authentication, rate limiting, and IP handling
type SecurityMiddleware struct {
	apiKey    string
	rateLimit *RateLimiter
}

// SecurityConfig holds configuration for security middleware
type SecurityConfig struct {
	APIKey         string
	RateLimitRPS   float64
	RateLimitBurst int
	PublicPaths    []string
}

// NewSecurityMiddleware creates a new consolidated security middleware
func NewSecurityMiddleware(cfg SecurityConfig) *SecurityMiddleware {
	return &SecurityMiddleware{
		apiKey:    cfg.APIKey,
		rateLimit: NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst),
	}
}

// Middleware returns a consolidated HTTP middleware that handles:
// - API key authentication (for protected routes)
// - Rate limiting (per IP)
// - Public route bypass
func (s *SecurityMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Apply rate limiting first (applies to all requests)
			s.rateLimit.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Apply authentication for protected routes
				s.applyAuthentication(next, w, r)
			})).ServeHTTP(w, r)
		})
	}
}

// applyAuthentication handles API key authentication with public route bypass
func (s *SecurityMiddleware) applyAuthentication(next http.Handler, w http.ResponseWriter, r *http.Request) {
	// Skip authentication if API key is not configured
	if s.apiKey == "" {
		next.ServeHTTP(w, r)
		return
	}

	// Get API key from header
	providedKey := r.Header.Get("X-API-Key")
	if providedKey == "" {
		slog.WarnContext(r.Context(), "API key missing",
			"ip", GetClientIP(r),
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.unauthorized(w, "API key required")
		return
	}

	// Constant-time comparison to prevent timing attacks
	if !ConstantTimeCompare(providedKey, s.apiKey) {
		slog.WarnContext(r.Context(), "Invalid API key",
			"ip", GetClientIP(r),
			"method", r.Method,
			"path", r.URL.Path,
		)
		s.unauthorized(w, "Invalid API key")
		return
	}

	// API key is valid
	next.ServeHTTP(w, r)
}

// unauthorized sends a 401 Unauthorized response
func (s *SecurityMiddleware) unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "API-Key")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized","message":"` + message + `"}`))
}
