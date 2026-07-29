package middleware

import "net/http"

func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			if origin != "" && origin == allowedOrigin {
				writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				writer.Header().Set("Vary", "Origin")
				writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			}
			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}
