package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("request panic", "request_id", RequestIDFromContext(request.Context()), "panic", recovered)
					response.Error(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan internal")
				}
			}()
			next.ServeHTTP(writer, request)
		})
	}
}
