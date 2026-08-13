package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

type TokenVerifier interface {
	Verify(context.Context, string) (domain.Identity, string, error)
}

type SessionValidator interface {
	Validate(context.Context, uuid.UUID, string) error
}

func Authenticate(tokens TokenVerifier, sessions SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			raw, ok := bearerToken(request.Header.Get("Authorization"))
			if !ok {
				response.FromError(writer, domain.ErrInvalidToken)
				return
			}
			identity, fingerprint, err := tokens.Verify(request.Context(), raw)
			if err != nil {
				response.FromError(writer, domain.ErrInvalidToken)
				return
			}
			if err := sessions.Validate(request.Context(), identity.UserID, fingerprint); err != nil {
				response.FromError(writer, domain.ErrSessionInvalid)
				return
			}
			next.ServeHTTP(writer, request.WithContext(WithIdentity(request.Context(), identity)))
		})
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
