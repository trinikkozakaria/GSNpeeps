package password

import (
	"testing"
	"time"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerify(t *testing.T) {
	hasher := New(config.Auth{
		ArgonMemoryKiB:   8 * 1024,
		ArgonIterations:  1,
		ArgonParallelism: 1,
		RequestWindow:    time.Minute,
	})
	encoded, err := hasher.Hash("synthetic-password")
	require.NoError(t, err)

	matches, err := hasher.Verify("synthetic-password", encoded)
	require.NoError(t, err)
	require.True(t, matches)

	matches, err = hasher.Verify("wrong-password", encoded)
	require.NoError(t, err)
	require.False(t, matches)
}
