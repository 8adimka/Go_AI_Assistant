package httpx

import (
	"fmt"
	"log/slog"
	"net/http"
)

func Recovery() func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					err, ok := v.(error)
					if !ok {
						err = fmt.Errorf("%v", v)
					}

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					slog.ErrorContext(r.Context(), "HTTP handler recovered from panic", "error", err)
				}
			}()

			handler.ServeHTTP(w, r)
		})
	}
}
