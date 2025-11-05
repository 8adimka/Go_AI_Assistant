package httpx

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

type statusAwareResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusAwareResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func Logger() func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw := &statusAwareResponseWriter{ResponseWriter: w}

			defer func() {
				// Get trace ID from context
				span := trace.SpanFromContext(r.Context())
				traceID := span.SpanContext().TraceID().String()

				logAttrs := []any{
					"http_method", r.Method,
					"http_path", r.URL.Path,
					"http_status", saw.status,
					"trace_id", traceID,
				}

				// Add user agent if available
				if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
					logAttrs = append(logAttrs, "http_user_agent", userAgent)
				}

				// Add remote address
				logAttrs = append(logAttrs, "http_remote_addr", r.RemoteAddr)

				if saw.status/100 == 5 {
					slog.ErrorContext(r.Context(), "HTTP request failed", logAttrs...)
				} else {
					slog.InfoContext(r.Context(), "HTTP request complete", logAttrs...)
				}
			}()

			handler.ServeHTTP(saw, r)
		})
	}
}

// LogWithTrace adds trace ID to log attributes
func LogWithTrace(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()

	// Add trace ID to attributes
	allAttrs := append([]any{"trace_id", traceID}, attrs...)

	switch level {
	case slog.LevelDebug:
		slog.DebugContext(ctx, msg, allAttrs...)
	case slog.LevelInfo:
		slog.InfoContext(ctx, msg, allAttrs...)
	case slog.LevelWarn:
		slog.WarnContext(ctx, msg, allAttrs...)
	case slog.LevelError:
		slog.ErrorContext(ctx, msg, allAttrs...)
	}
}
