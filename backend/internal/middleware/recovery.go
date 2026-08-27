package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					// Stack trace di-log agar panic dapat ditelusuri; isinya tidak pernah
					// dikirim ke client.
					logger.Error("request panic",
						"request_id", RequestIDFromContext(request.Context()),
						"method", request.Method,
						"path", request.URL.Path,
						"panic", recovered,
						"stack", string(debug.Stack()),
					)
					response.Error(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan internal")
				}
			}()
			next.ServeHTTP(writer, request)
		})
	}
}
