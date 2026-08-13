package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (writer *statusWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			wrapped := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
			next.ServeHTTP(wrapped, request)
			logger.Info("http request",
				"request_id", RequestIDFromContext(request.Context()),
				"method", request.Method,
				"path", request.URL.Path,
				"status", wrapped.status,
				"duration_ms", time.Since(started).Milliseconds(),
				"remote_addr", request.RemoteAddr,
			)
		})
	}
}
