package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusResponseWriter captures the status code so the request line can report
// it without a second log call.
type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// LoggingMiddleware emits one Debug line per request. It replaces the previous
// pair of unleveled lines per request, which doubled the API's log volume for
// no added information.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			// Skip logging for health check endpoint
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		slog.Debug("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
