package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSessionStore membutuhkan Redis nyata (TEST_REDIS_URL) karena SCAN, EXISTS, dan
// TTL tidak dapat dipalsukan tanpa server. Repo tidak memakai miniredis (bukan bagian
// baseline dependency).
func newTestSessionStore(t *testing.T) (*SessionStore, *goredis.Client) {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL tidak diset; test session store dilewati")
	}
	opts, err := goredis.ParseURL(url)
	require.NoError(t, err)
	client := goredis.NewClient(opts)
	require.NoError(t, client.Ping(context.Background()).Err())
	t.Cleanup(func() { _ = client.Close() })
	return NewSessionStore(client), client
}

func TestSessionStoreKeepsConcurrentTokensValid(t *testing.T) {
	store, _ := newTestSessionStore(t)
	ctx := context.Background()
	userID := uuid.New()
	deviceA, deviceB := "fp-"+uuid.NewString(), "fp-"+uuid.NewString()
	t.Cleanup(func() { _ = store.Revoke(ctx, userID) })

	require.NoError(t, store.Save(ctx, userID, deviceA, time.Hour))
	require.NoError(t, store.Save(ctx, userID, deviceB, time.Hour))

	// Login kedua tidak menimpa yang pertama.
	require.NoError(t, store.Validate(ctx, userID, deviceA))
	require.NoError(t, store.Validate(ctx, userID, deviceB))
}

func TestSessionStoreRevokeTokenOnlyDropsOneDevice(t *testing.T) {
	store, _ := newTestSessionStore(t)
	ctx := context.Background()
	userID := uuid.New()
	deviceA, deviceB := "fp-"+uuid.NewString(), "fp-"+uuid.NewString()
	t.Cleanup(func() { _ = store.Revoke(ctx, userID) })

	require.NoError(t, store.Save(ctx, userID, deviceA, time.Hour))
	require.NoError(t, store.Save(ctx, userID, deviceB, time.Hour))

	require.NoError(t, store.RevokeToken(ctx, userID, deviceA))

	assert.ErrorIs(t, store.Validate(ctx, userID, deviceA), domain.ErrSessionInvalid, "device A logout")
	assert.NoError(t, store.Validate(ctx, userID, deviceB), "device B tetap berlaku")
}

func TestSessionStoreRevokeDropsEveryDevice(t *testing.T) {
	store, _ := newTestSessionStore(t)
	ctx := context.Background()
	userID := uuid.New()
	devices := []string{"fp-" + uuid.NewString(), "fp-" + uuid.NewString(), "fp-" + uuid.NewString()}
	t.Cleanup(func() { _ = store.Revoke(ctx, userID) })

	for _, fp := range devices {
		require.NoError(t, store.Save(ctx, userID, fp, time.Hour))
	}

	require.NoError(t, store.Revoke(ctx, userID))

	for _, fp := range devices {
		assert.ErrorIsf(t, store.Validate(ctx, userID, fp), domain.ErrSessionInvalid, "device %s dicabut oleh security revoke", fp)
	}
}

func TestSessionStoreSaveSetsPerKeyTTL(t *testing.T) {
	store, client := newTestSessionStore(t)
	ctx := context.Background()
	userID := uuid.New()
	fp := "fp-" + uuid.NewString()
	t.Cleanup(func() { _ = store.Revoke(ctx, userID) })

	require.NoError(t, store.Save(ctx, userID, fp, 45*time.Minute))

	ttl, err := client.TTL(ctx, sessionKey(userID, fp)).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0), "key wajib punya TTL agar entri basi bersih sendiri")
	assert.LessOrEqual(t, ttl, 45*time.Minute)
}

func TestSessionStoreRevokeIsSafeWhenUserHasNoSessions(t *testing.T) {
	store, _ := newTestSessionStore(t)
	assert.NoError(t, store.Revoke(context.Background(), uuid.New()))
}
