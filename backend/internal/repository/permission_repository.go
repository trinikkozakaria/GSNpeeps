package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5"
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

// RolePermissions mengembalikan seluruh kapabilitas satu role sebagai peta `modul.aksi`.
// Dipakai cache untuk mengisi ulang satu role sekaligus, bukan per pemeriksaan.
func (r *PermissionRepository) RolePermissions(
	ctx context.Context,
	role domain.RoleName,
) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT p.modul, p.aksi, p.diizinkan
        FROM permissions p
        JOIN roles r ON r.id = p.role_id
        WHERE r.nama = $1
    `, string(role))
	if err != nil {
		return nil, fmt.Errorf("list role permissions: %w", err)
	}
	defer rows.Close()

	capabilities := map[string]bool{}
	for rows.Next() {
		var module, action string
		var allowed bool
		if err := rows.Scan(&module, &action, &allowed); err != nil {
			return nil, fmt.Errorf("scan role permission: %w", err)
		}
		capabilities[module+"."+action] = allowed
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate role permissions: %w", err)
	}
	return capabilities, nil
}

// ListRoles mengembalikan empat role sistem berurut nama.
func (r *PermissionRepository) ListRoles(ctx context.Context) ([]domain.RoleSummary, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, nama FROM roles ORDER BY nama`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	items := []domain.RoleSummary{}
	for rows.Next() {
		var item domain.RoleSummary
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		item.Description = domain.RoleDescription(item.Name)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}
	return items, nil
}

// ListPermissions mengembalikan matriks lengkap empat role (D-034).
func (r *PermissionRepository) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT p.id, p.role_id, p.modul, p.aksi, p.diizinkan
        FROM permissions p
        JOIN roles r ON r.id = p.role_id
        ORDER BY r.nama, p.modul, p.aksi
    `)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()

	items := []domain.Permission{}
	for rows.Next() {
		var item domain.Permission
		if err := rows.Scan(
			&item.ID, &item.RoleID, &item.Module, &item.Action, &item.IsAllowed,
		); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permissions: %w", err)
	}
	return items, nil
}

// FindRoleName mengembalikan nama role. ErrNotFound bila role_id tidak dikenal.
func (r *PermissionRepository) FindRoleName(
	ctx context.Context,
	roleID uuid.UUID,
) (domain.RoleName, error) {
	var name domain.RoleName
	err := executor(ctx, r.pool).QueryRow(ctx,
		`SELECT nama FROM roles WHERE id = $1`, roleID,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find role: %w", err)
	}
	return name, nil
}

// CurrentPermission membaca nilai sebelum perubahan untuk audit before/after. Baris yang
// belum ada dianggap tidak diizinkan.
func (r *PermissionRepository) CurrentPermission(
	ctx context.Context,
	change domain.PermissionChange,
) (bool, error) {
	var allowed bool
	err := executor(ctx, r.pool).QueryRow(ctx, `
        SELECT diizinkan FROM permissions
        WHERE role_id = $1 AND modul = $2 AND aksi = $3
    `, change.RoleID, change.Module, change.Action).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read current permission: %w", err)
	}
	return allowed, nil
}

// UpsertPermission menulis satu baris matriks dan mengembalikan ID-nya. Upsert dipakai agar
// kapabilitas katalog yang belum pernah di-seed tetap dapat diatur tanpa migration tambahan.
func (r *PermissionRepository) UpsertPermission(
	ctx context.Context,
	change domain.PermissionChange,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
        INSERT INTO permissions (role_id, modul, aksi, diizinkan)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (role_id, modul, aksi) DO UPDATE SET diizinkan = EXCLUDED.diizinkan
        RETURNING id
    `, change.RoleID, change.Module, change.Action, change.IsAllowed).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert permission: %w", err)
	}
	return id, nil
}
