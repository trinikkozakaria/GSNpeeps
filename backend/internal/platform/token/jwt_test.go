package token

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestIssueAndVerify(t *testing.T) {
	manager := New(config.JWT{Secret: strings.Repeat("s", 32), TTL: 8 * time.Hour})
	expected := domain.Identity{
		UserID:     uuid.New(),
		EmployeeID: uuid.New(),
		Role:       domain.RoleHR,
	}
	raw, sessionFingerprint, ttl, err := manager.Issue(expected)
	require.NoError(t, err)
	require.Equal(t, 8*time.Hour, ttl)
	require.NotEmpty(t, sessionFingerprint)

	actual, actualFingerprint, err := manager.Verify(context.Background(), raw)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, sessionFingerprint, actualFingerprint)
}
