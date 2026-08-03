package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

// PermissionStore adalah kebutuhan administrasi matriks permission dari sisi service.
type PermissionStore interface {
	ListRoles(context.Context) ([]domain.RoleSummary, error)
	ListPermissions(context.Context) ([]domain.Permission, error)
	FindRoleName(context.Context, uuid.UUID) (domain.RoleName, error)
	CurrentPermission(context.Context, domain.PermissionChange) (bool, error)
	UpsertPermission(context.Context, domain.PermissionChange) (uuid.UUID, error)
}

// AuditReader membaca Audit Log. Boundary ini sengaja tidak memiliki update maupun delete.
type AuditReader interface {
	List(context.Context, domain.AuditLogFilter) (domain.AuditLogPage, error)
}

// PermissionCache menyimpan kapabilitas per role. Invalidasi dipanggil setiap kali matriks
// berubah agar keputusan otorisasi tidak memakai nilai lama.
type PermissionCache interface {
	Invalidate(context.Context, domain.RoleName) error
}

// AccessService melayani modul AKSES: daftar role, matriks permission, dan Audit Log.
type AccessService struct {
	permissions PermissionStore
	audits      AuditReader
	tx          EmployeeTransactionManager
	audit       AuditWriter
	cache       PermissionCache
	now         func() time.Time
}

func NewAccessService(
	permissions PermissionStore,
	audits AuditReader,
	tx EmployeeTransactionManager,
	audit AuditWriter,
	cache PermissionCache,
) *AccessService {
	return &AccessService{
		permissions: permissions,
		audits:      audits,
		tx:          tx,
		audit:       audit,
		cache:       cache,
		now:         time.Now,
	}
}

// canReadAccess membatasi seluruh pembacaan modul AKSES ke HR dan Top Management. Karyawan
// dan Atasan tetap menerima 403 pada akses langsung meskipun menunya disembunyikan frontend.
func canReadAccess(identity domain.Identity) error {
	if identity.Role != domain.RoleHR && identity.Role != domain.RoleTopManagement {
		return domain.ErrForbidden
	}
	return nil
}

func (s *AccessService) ListRoles(
	ctx context.Context,
	identity domain.Identity,
) ([]domain.RoleSummary, error) {
	if err := canReadAccess(identity); err != nil {
		return nil, err
	}
	roles, err := s.permissions.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	return roles, nil
}

func (s *AccessService) PermissionMatrix(
	ctx context.Context,
	identity domain.Identity,
) ([]domain.Permission, error) {
	if err := canReadAccess(identity); err != nil {
		return nil, err
	}
	items, err := s.permissions.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	return items, nil
}

// UpdatePermission mengubah satu sel matriks. Hanya HR yang boleh; Top Management selalu 403
// sehingga tidak ada jalur bagi role read-only untuk memperluas kewenangannya sendiri.
//
// Seluruh langkah berada dalam satu transaction: role diverifikasi, invariant produk
// diperiksa, nilai lama dibaca untuk audit, baris ditulis, audit before/after ditulis, lalu
// cache kapabilitas role tersebut dibatalkan. Kegagalan invalidasi membatalkan perubahan agar
// tidak ada jendela ketika database dan cache berbeda.
func (s *AccessService) UpdatePermission(
	ctx context.Context,
	identity domain.Identity,
	change domain.PermissionChange,
	meta RequestMeta,
) (uuid.UUID, error) {
	if identity.Role != domain.RoleHR {
		return uuid.Nil, domain.ErrForbidden
	}

	var permissionID uuid.UUID
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		roleName, err := s.permissions.FindRoleName(txContext, change.RoleID)
		if errors.Is(err, repository.ErrNotFound) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := domain.ValidatePermissionChange(roleName, change); err != nil {
			return err
		}

		before, err := s.permissions.CurrentPermission(txContext, change)
		if err != nil {
			return err
		}
		permissionID, err = s.permissions.UpsertPermission(txContext, change)
		if err != nil {
			return err
		}
		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "PERMISSION_UPDATE",
			Module: "akses",
			DataID: &permissionID,
			Detail: map[string]any{
				"role":       string(roleName),
				"modul":      change.Module,
				"aksi":       change.Action,
				"sebelum":    before,
				"sesudah":    change.IsAllowed,
				"request_id": meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		}); err != nil {
			return err
		}
		return s.cache.Invalidate(txContext, roleName)
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("update permission: %w", err)
	}
	return permissionID, nil
}

// ListAuditLogs mengembalikan Audit Log yang sudah di-redact untuk HR dan Top Management.
func (s *AccessService) ListAuditLogs(
	ctx context.Context,
	identity domain.Identity,
	filter domain.AuditLogFilter,
) (domain.AuditLogPage, error) {
	if err := canReadAccess(identity); err != nil {
		return domain.AuditLogPage{}, err
	}
	filter.Page, filter.Limit = normalizePaging(filter.Page, filter.Limit)

	page, err := s.audits.List(ctx, filter)
	if err != nil {
		return domain.AuditLogPage{}, fmt.Errorf("list audit logs: %w", err)
	}
	for index := range page.Items {
		page.Items[index].Detail = domain.RedactAuditDetail(page.Items[index].Detail)
	}
	return page, nil
}
