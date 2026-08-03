package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// PermissionSource membaca kapabilitas role dari sumber kebenaran, yaitu PostgreSQL.
type PermissionSource interface {
	RolePermissions(context.Context, domain.RoleName) (map[string]bool, error)
}

// PermissionCacheStore adalah cache kapabilitas role.
type PermissionCacheStore interface {
	Load(context.Context, domain.RoleName) (map[string]bool, bool, error)
	Store(context.Context, domain.RoleName, map[string]bool) error
	Invalidate(context.Context, domain.RoleName) error
}

// CachedPermissionChecker menjawab pemeriksaan permission middleware dengan cache di depan
// database.
//
// Cache tidak pernah dapat memperluas kewenangan: kegagalan Redis membuat pemeriksaan jatuh
// ke database yang merupakan sumber kebenaran, sedangkan perubahan matriks membatalkan entri
// role terkait di dalam transaction perubahan. Karena permission tidak berada di dalam JWT,
// perubahan berlaku tanpa menunggu token kedaluwarsa.
type CachedPermissionChecker struct {
	source PermissionSource
	cache  PermissionCacheStore
	logger *slog.Logger
}

func NewCachedPermissionChecker(
	source PermissionSource,
	cache PermissionCacheStore,
	logger *slog.Logger,
) *CachedPermissionChecker {
	return &CachedPermissionChecker{source: source, cache: cache, logger: logger}
}

func (c *CachedPermissionChecker) HasPermission(
	ctx context.Context,
	role domain.RoleName,
	module string,
	action string,
) (bool, error) {
	capability := module + "." + action

	capabilities, hit, err := c.cache.Load(ctx, role)
	if err != nil {
		// Cache tidak tersedia bukan alasan mengizinkan maupun menolak secara sepihak;
		// pemeriksaan dilanjutkan ke database.
		c.logger.WarnContext(ctx, "permission cache unavailable", "error", err)
	}
	if hit {
		return capabilities[capability], nil
	}

	capabilities, err = c.source.RolePermissions(ctx, role)
	if err != nil {
		return false, fmt.Errorf("load role permissions: %w", err)
	}
	if err := c.cache.Store(ctx, role, capabilities); err != nil {
		c.logger.WarnContext(ctx, "permission cache write failed", "error", err)
	}
	return capabilities[capability], nil
}
