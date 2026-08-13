package redis

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

type SessionStore struct {
	client *goredis.Client
}

func NewSessionStore(client *goredis.Client) *SessionStore {
	return &SessionStore{client: client}
}

func (s *SessionStore) Save(ctx context.Context, userID uuid.UUID, fingerprint string, ttl time.Duration) error {
	if err := s.client.Set(ctx, sessionKey(userID), fingerprint, ttl).Err(); err != nil {
		return fmt.Errorf("save active session: %w", err)
	}
	return nil
}

func (s *SessionStore) Validate(ctx context.Context, userID uuid.UUID, fingerprint string) error {
	stored, err := s.client.Get(ctx, sessionKey(userID)).Result()
	if errors.Is(err, goredis.Nil) {
		return domain.ErrSessionInvalid
	}
	if err != nil {
		return fmt.Errorf("read active session: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(fingerprint)) != 1 {
		return domain.ErrSessionInvalid
	}
	return nil
}

func (s *SessionStore) Revoke(ctx context.Context, userID uuid.UUID) error {
	if err := s.client.Del(ctx, sessionKey(userID)).Err(); err != nil {
		return fmt.Errorf("revoke active session: %w", err)
	}
	return nil
}

func sessionKey(userID uuid.UUID) string {
	return "session:" + userID.String()
}
