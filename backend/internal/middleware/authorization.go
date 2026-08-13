package middleware

import (
	"context"
	"net/http"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
)

type PermissionChecker interface {
	HasPermission(context.Context, domain.RoleName, string, string) (bool, error)
}

func RequirePermission(
	checker PermissionChecker,
	module string,
	action string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			identity, ok := IdentityFromContext(request.Context())
			if !ok {
				response.FromError(writer, domain.ErrInvalidToken)
				return
			}
			allowed, err := checker.HasPermission(request.Context(), identity.Role, module, action)
			if err != nil {
				response.FromError(writer, err)
				return
			}
			if !allowed {
				response.FromError(writer, domain.ErrForbidden)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}
