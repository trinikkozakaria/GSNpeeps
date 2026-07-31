package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

// AccessTokenCookieName is the cookie the frontend receives on login and
// the browser sends back automatically on every request to this API.
const AccessTokenCookieName = "gsnpeeps_token"

type TokenVerifier interface {
	Verify(context.Context, string) (domain.Identity, string, error)
}

type SessionValidator interface {
	Validate(context.Context, uuid.UUID, string) error
}

func Authenticate(tokens TokenVerifier, sessions SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			raw, ok := tokenFromCookie(request)
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

func tokenFromCookie(request *http.Request) (string, bool) {
	cookie, err := request.Cookie(AccessTokenCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}
