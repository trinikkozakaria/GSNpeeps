package middleware

import (
	"context"
	"net/http"
	"regexp"

	"github.com/google/uuid"
)

type contextKey string

const (
	requestIDKey    contextKey = "request_id"
	requestIDHeader            = "X-Request-ID"
)

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get(requestIDHeader)
		if !validRequestID.MatchString(requestID) {
			requestID = uuid.NewString()
		}
		writer.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(request.Context(), requestIDKey, requestID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}
