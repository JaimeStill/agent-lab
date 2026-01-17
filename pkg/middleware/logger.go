package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger returns middleware that logs HTTP requests with method, URI, remote address, and duration.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info(
				"request",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
				"addr", r.RemoteAddr,
				"duration", time.Since(start),
			)
		})
	}
}
