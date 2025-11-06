package httpx

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter provides per-IP rate limiting
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter with the given requests per second and burst
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// getLimiter returns the rate limiter for a given IP address
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// Middleware returns an HTTP middleware that enforces rate limiting per IP
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP (handle X-Forwarded-For and X-Real-IP)
			ip := GetClientIP(r)

			limiter := rl.getLimiter(ip)

			if !limiter.Allow() {
				slog.WarnContext(r.Context(), "Rate limit exceeded",
					"ip", ip,
					"method", r.Method,
					"path", r.URL.Path,
					"user_agent", r.UserAgent(),
				)

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", rl.rps))
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded","message":"too many requests, please try again later"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP extracts the client IP from the request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, use the first one
		for idx := 0; idx < len(xff); idx++ {
			if xff[idx] == ',' {
				return xff[:idx]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
