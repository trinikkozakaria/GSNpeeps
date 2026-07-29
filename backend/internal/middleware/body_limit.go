package middleware

import "net/http"

func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Body != nil {
				request.Body = http.MaxBytesReader(writer, request.Body, maxBytes)
			}
			next.ServeHTTP(writer, request)
		})
	}
}
