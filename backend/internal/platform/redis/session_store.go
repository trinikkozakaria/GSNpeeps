package redis

import (
	"context"
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
	if err := s.client.Set(ctx, sessionKey(userID, fingerprint), "1", ttl).Err(); err != nil {
		return fmt.Errorf("save active session: %w", err)
	}
	return nil
}

func (s *SessionStore) Validate(ctx context.Context, userID uuid.UUID, fingerprint string) error {
	exists, err := s.client.Exists(ctx, sessionKey(userID, fingerprint)).Result()
	if errors.Is(err, goredis.Nil) || exists == 0 {
		return domain.ErrSessionInvalid
	}
	if err != nil {
		return fmt.Errorf("read active session: %w", err)
	}
	return nil
}

func (s *SessionStore) Revoke(ctx context.Context, userID uuid.UUID) error {
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, "session:"+userID.String()+":*", 100).Result()
		if err != nil {
			return fmt.Errorf("scan active sessions: %w", err)
		}
		if len(keys) > 0 {
			if err := s.client.Unlink(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("revoke active sessions: %w", err)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (s *SessionStore) RevokeToken(ctx context.Context, userID uuid.UUID, fingerprint string) error {
	if err := s.client.Del(ctx, sessionKey(userID, fingerprint)).Err(); err != nil {
		return fmt.Errorf("revoke active session: %w", err)
	}
	return nil
}

func sessionKey(userID uuid.UUID, fingerprint string) string {
	return "session:" + userID.String() + ":" + fingerprint
}
