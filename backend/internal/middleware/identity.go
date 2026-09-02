package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

const identityKey contextKey = "identity"
const sessionFingerprintKey contextKey = "session_fingerprint"

type Identity = domain.Identity

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

func WithSessionFingerprint(ctx context.Context, fingerprint string) context.Context {
	return context.WithValue(ctx, sessionFingerprintKey, fingerprint)
}
func SessionFingerprintFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(sessionFingerprintKey).(string)
	return value, ok && value != ""
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey).(Identity)
	if !ok || identity.UserID == uuid.Nil || identity.EmployeeID == uuid.Nil || !identity.Role.Valid() {
		return Identity{}, false
	}
	return identity, true
}
