package middleware
 
import (
	"net/http"
	"strings"
)
 
func CORS(allowedOrigins string) func(http.Handler) http.Handler {
	origins := make(map[string]bool)
	for _, o := range strings.Split(allowedOrigins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = true
		}
	}
 
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			if origin != "" && origins[origin] {
				writer.Header().Set("Access-Control-Allow-Origin", origin)
				writer.Header().Set("Vary", "Origin")
				writer.Header().Set("Access-Control-Allow-Credentials", "true")
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
 