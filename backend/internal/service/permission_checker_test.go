package service

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"testing"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type permissionSourceStub struct {
	capabilities map[domain.RoleName]map[string]bool
	loads        int
	err          error
}

func (s *permissionSourceStub) RolePermissions(
	_ context.Context, role domain.RoleName,
) (map[string]bool, error) {
	s.loads++
	if s.err != nil {
		return nil, s.err
	}
	return s.capabilities[role], nil
}

// permissionCacheStoreStub meniru Redis dengan kemungkinan gangguan pada baca dan tulis.
type permissionCacheStoreStub struct {
	entries  map[domain.RoleName]map[string]bool
	loadErr  error
	storeErr error
}

func newPermissionCacheStoreStub() *permissionCacheStoreStub {
	return &permissionCacheStoreStub{entries: map[domain.RoleName]map[string]bool{}}
}

func (s *permissionCacheStoreStub) Load(
	_ context.Context, role domain.RoleName,
) (map[string]bool, bool, error) {
	if s.loadErr != nil {
		return nil, false, s.loadErr
	}
	entry, ok := s.entries[role]
	return entry, ok, nil
}

func (s *permissionCacheStoreStub) Store(
	_ context.Context, role domain.RoleName, capabilities map[string]bool,
) error {
	if s.storeErr != nil {
		return s.storeErr
	}
	// Redis menyimpan salinan terserialisasi; stub menyalin agar perubahan pada sumber tidak
	// terlihat melalui cache tanpa invalidasi.
	stored := make(map[string]bool, len(capabilities))
	maps.Copy(stored, capabilities)
	s.entries[role] = stored
	return nil
}

func (s *permissionCacheStoreStub) Invalidate(_ context.Context, role domain.RoleName) error {
	delete(s.entries, role)
	return nil
}

func newCheckerForTest() (*CachedPermissionChecker, *permissionSourceStub, *permissionCacheStoreStub) {
	source := &permissionSourceStub{capabilities: map[domain.RoleName]map[string]bool{
		domain.RoleHR: {"akses.read": true, "akses.update": true, "audit.read": true},
		domain.RoleTopManagement: {
			"akses.read": true, "akses.update": false, "audit.read": true,
		},
		domain.RoleEmployee: {"akses.read": false, "audit.read": false},
	}}
	cache := newPermissionCacheStoreStub()
	return NewCachedPermissionChecker(source, cache, slog.New(slog.DiscardHandler)), source, cache
}

func TestHasPermissionCachesRoleCapabilities(t *testing.T) {
	checker, source, _ := newCheckerForTest()

	first, err := checker.HasPermission(
		context.Background(), domain.RoleHR, domain.ModuleAccess, domain.ActionUpdate,
	)
	require.NoError(t, err)
	second, err := checker.HasPermission(
		context.Background(), domain.RoleHR, domain.ModuleAudit, domain.ActionRead,
	)
	require.NoError(t, err)

	assert.True(t, first)
	assert.True(t, second)
	// Satu role dimuat sekali; pemeriksaan berikutnya dilayani cache.
	assert.Equal(t, 1, source.loads)
}

// Perubahan matriks berlaku seketika setelah invalidasi, tanpa menunggu JWT kedaluwarsa.
func TestHasPermissionSeesChangeAfterInvalidate(t *testing.T) {
	checker, source, cache := newCheckerForTest()

	allowed, err := checker.HasPermission(
		context.Background(), domain.RoleTopManagement, domain.ModuleAccess, domain.ActionUpdate,
	)
	require.NoError(t, err)
	require.False(t, allowed)

	source.capabilities[domain.RoleTopManagement]["akses.read"] = false
	require.NoError(t, cache.Invalidate(context.Background(), domain.RoleTopManagement))

	allowed, err = checker.HasPermission(
		context.Background(), domain.RoleTopManagement, domain.ModuleAccess, domain.ActionRead,
	)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 2, source.loads)
}

// Cache lama tetap dipakai sampai dibatalkan; ini alasan invalidasi berada di dalam
// transaction perubahan permission.
func TestHasPermissionKeepsCachedValueUntilInvalidated(t *testing.T) {
	checker, source, _ := newCheckerForTest()

	_, err := checker.HasPermission(
		context.Background(), domain.RoleHR, domain.ModuleAccess, domain.ActionUpdate,
	)
	require.NoError(t, err)
	source.capabilities[domain.RoleHR]["akses.update"] = false

	allowed, err := checker.HasPermission(
		context.Background(), domain.RoleHR, domain.ModuleAccess, domain.ActionUpdate,
	)
	require.NoError(t, err)
	assert.True(t, allowed)
}

// Cache yang tidak tersedia tidak boleh mengubah hasil otorisasi; database tetap dibaca.
func TestHasPermissionFallsBackToDatabaseWhenCacheFails(t *testing.T) {
	checker, source, cache := newCheckerForTest()
	cache.loadErr = errors.New("redis tidak tersedia")
	cache.storeErr = errors.New("redis tidak tersedia")

	allowed, err := checker.HasPermission(
		context.Background(), domain.RoleEmployee, domain.ModuleAudit, domain.ActionRead,
	)
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = checker.HasPermission(
		context.Background(), domain.RoleHR, domain.ModuleAudit, domain.ActionRead,
	)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 2, source.loads)
}

// Kapabilitas yang tidak tercatat berarti tidak diizinkan, bukan diizinkan secara default.
func TestHasPermissionDeniesUnknownCapability(t *testing.T) {
	checker, _, _ := newCheckerForTest()

	allowed, err := checker.HasPermission(
		context.Background(), domain.RoleEmployee, domain.ModuleEmployee, domain.ActionDelete,
	)

	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestHasPermissionPropagatesDatabaseFailure(t *testing.T) {
	checker, source, _ := newCheckerForTest()
	source.err = errors.New("koneksi putus")

	_, err := checker.HasPermission(
		context.Background(), domain.RoleHR, domain.ModuleAccess, domain.ActionRead,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "load role permissions")
}
