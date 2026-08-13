package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

// RateLimiter is the minimum boundary required by login and per-user policies.
// The concrete Redis policy is added together with the endpoint that defines its key.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

func AuthenticatedRateLimit(
	limiter RateLimiter,
	limit int,
	window time.Duration,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			identity, ok := IdentityFromContext(request.Context())
			if !ok {
				response.FromError(writer, domain.ErrInvalidToken)
				return
			}
			allowed, err := limiter.Allow(
				request.Context(),
				"user:"+identity.UserID.String(),
				limit,
				window,
			)
			if err != nil {
				response.FromError(writer, err)
				return
			}
			if !allowed {
				response.FromError(writer, domain.ErrRateLimited)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}
