package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

const identityKey contextKey = "identity"

type Identity = domain.Identity

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey).(Identity)
	if !ok || identity.UserID == uuid.Nil || identity.EmployeeID == uuid.Nil || !identity.Role.Valid() {
		return Identity{}, false
	}
	return identity, true
}
