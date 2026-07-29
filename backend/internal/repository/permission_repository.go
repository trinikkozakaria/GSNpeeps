package repository

import (
	"context"
	"fmt"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PermissionRepository struct {
	pool *pgxpool.Pool
}

func NewPermissionRepository(pool *pgxpool.Pool) *PermissionRepository {
	return &PermissionRepository{pool: pool}
}

func (r *PermissionRepository) HasPermission(
	ctx context.Context,
	role domain.RoleName,
	module string,
	action string,
) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx, `
        SELECT COALESCE(BOOL_OR(p.diizinkan), FALSE)
        FROM permissions p
        JOIN roles r ON r.id = p.role_id
        WHERE r.nama = $1
          AND p.modul = $2
          AND p.aksi = $3
    `, role, module, action).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check permission: %w", err)
	}
	return allowed, nil
}
