package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type permissionStoreStub struct {
	roleNames map[uuid.UUID]domain.RoleName
	current   bool
	upserted  []domain.PermissionChange
	roles     []domain.RoleSummary
	matrix    []domain.Permission
	upsertErr error
}

func (s *permissionStoreStub) ListRoles(context.Context) ([]domain.RoleSummary, error) {
	return s.roles, nil
}

func (s *permissionStoreStub) ListPermissions(context.Context) ([]domain.Permission, error) {
	return s.matrix, nil
}

func (s *permissionStoreStub) FindRoleName(
	_ context.Context, roleID uuid.UUID,
) (domain.RoleName, error) {
	name, ok := s.roleNames[roleID]
	if !ok {
		return "", repository.ErrNotFound
	}
	return name, nil
}

func (s *permissionStoreStub) CurrentPermission(
	context.Context, domain.PermissionChange,
) (bool, error) {
	return s.current, nil
}

func (s *permissionStoreStub) UpsertPermission(
	_ context.Context, change domain.PermissionChange,
) (uuid.UUID, error) {
	if s.upsertErr != nil {
		return uuid.Nil, s.upsertErr
	}
	s.upserted = append(s.upserted, change)
	return uuid.New(), nil
}

type auditReaderStub struct {
	filter domain.AuditLogFilter
	page   domain.AuditLogPage
}

func (s *auditReaderStub) List(
	_ context.Context, filter domain.AuditLogFilter,
) (domain.AuditLogPage, error) {
	s.filter = filter
	return s.page, nil
}

type permissionCacheStub struct {
	invalidated []domain.RoleName
	err         error
}

func (s *permissionCacheStub) Invalidate(_ context.Context, role domain.RoleName) error {
	if s.err != nil {
		return s.err
	}
	s.invalidated = append(s.invalidated, role)
	return nil
}

type auditRecorder struct{ entries []domain.AuditEntry }

func (r *auditRecorder) Append(_ context.Context, entry domain.AuditEntry) error {
	r.entries = append(r.entries, entry)
	return nil
}

type accessFixture struct {
	service     *AccessService
	permissions *permissionStoreStub
	audits      *auditReaderStub
	audit       *auditRecorder
	cache       *permissionCacheStub
	roleIDs     map[domain.RoleName]uuid.UUID
}

func newAccessFixture() accessFixture {
	roleIDs := map[domain.RoleName]uuid.UUID{}
	roleNames := map[uuid.UUID]domain.RoleName{}
	for _, role := range []domain.RoleName{
		domain.RoleEmployee, domain.RoleSupervisor, domain.RoleHR, domain.RoleTopManagement,
	} {
		id := uuid.New()
		roleIDs[role] = id
		roleNames[id] = role
	}

	permissions := &permissionStoreStub{roleNames: roleNames}
	audits := &auditReaderStub{}
	audit := &auditRecorder{}
	cache := &permissionCacheStub{}
	return accessFixture{
		service: NewAccessService(
			permissions, audits, transactionStub{}, audit, cache,
		),
		permissions: permissions,
		audits:      audits,
		audit:       audit,
		cache:       cache,
		roleIDs:     roleIDs,
	}
}

func identityFor(role domain.RoleName) domain.Identity {
	return domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: role}
}

func TestAccessReadsAreLimitedToHumanResources(t *testing.T) {
	fixture := newAccessFixture()

	for _, role := range []domain.RoleName{domain.RoleHR} {
		_, err := fixture.service.ListRoles(context.Background(), identityFor(role))
		assert.NoErrorf(t, err, "role %s harus dapat membaca daftar role", role)
		_, err = fixture.service.PermissionMatrix(context.Background(), identityFor(role))
		assert.NoErrorf(t, err, "role %s harus dapat membaca matriks", role)
		_, err = fixture.service.ListAuditLogs(
			context.Background(), identityFor(role), domain.AuditLogFilter{},
		)
		assert.NoErrorf(t, err, "role %s harus dapat membaca audit log", role)
	}

	for _, role := range []domain.RoleName{domain.RoleEmployee, domain.RoleSupervisor, domain.RoleTopManagement} {
		_, err := fixture.service.ListRoles(context.Background(), identityFor(role))
		assert.ErrorIsf(t, err, domain.ErrForbidden, "role %s tidak boleh membaca role", role)
		_, err = fixture.service.PermissionMatrix(context.Background(), identityFor(role))
		assert.ErrorIsf(t, err, domain.ErrForbidden, "role %s tidak boleh membaca matriks", role)
		_, err = fixture.service.ListAuditLogs(
			context.Background(), identityFor(role), domain.AuditLogFilter{},
		)
		assert.ErrorIsf(t, err, domain.ErrForbidden, "role %s tidak boleh membaca audit", role)
	}
}

func TestUpdatePermissionAppliesChangeAndInvalidatesCache(t *testing.T) {
	fixture := newAccessFixture()
	fixture.permissions.current = false
	change := domain.PermissionChange{
		RoleID:    fixture.roleIDs[domain.RoleSupervisor],
		Module:    domain.ModuleOvertime,
		Action:    domain.ActionApprove,
		IsAllowed: true,
	}

	id, err := fixture.service.UpdatePermission(
		context.Background(), identityFor(domain.RoleHR), change, RequestMeta{RequestID: "req-1"},
	)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
	require.Len(t, fixture.permissions.upserted, 1)
	assert.Equal(t, change, fixture.permissions.upserted[0])

	// Perubahan berlaku seketika karena cache role terkait dibatalkan.
	assert.Equal(t, []domain.RoleName{domain.RoleSupervisor}, fixture.cache.invalidated)

	require.Len(t, fixture.audit.entries, 1)
	entry := fixture.audit.entries[0]
	assert.Equal(t, "PERMISSION_UPDATE", entry.Action)
	assert.Equal(t, "akses", entry.Module)
	assert.Equal(t, false, entry.Detail["sebelum"])
	assert.Equal(t, true, entry.Detail["sesudah"])
	assert.Equal(t, string(domain.RoleSupervisor), entry.Detail["role"])
}

// Top Management read-only pada AKSES; penolakan tidak boleh menyentuh matriks.
func TestUpdatePermissionRejectsTopManagement(t *testing.T) {
	fixture := newAccessFixture()

	_, err := fixture.service.UpdatePermission(
		context.Background(), identityFor(domain.RoleTopManagement), domain.PermissionChange{
			RoleID:    fixture.roleIDs[domain.RoleTopManagement],
			Module:    domain.ModuleAccess,
			Action:    domain.ActionUpdate,
			IsAllowed: true,
		}, RequestMeta{},
	)

	assert.ErrorIs(t, err, domain.ErrForbidden)
	assert.Empty(t, fixture.permissions.upserted)
	assert.Empty(t, fixture.audit.entries)
	assert.Empty(t, fixture.cache.invalidated)
}

func TestUpdatePermissionRejectsKaryawanAndAtasan(t *testing.T) {
	for _, role := range []domain.RoleName{domain.RoleEmployee, domain.RoleSupervisor} {
		fixture := newAccessFixture()

		_, err := fixture.service.UpdatePermission(
			context.Background(), identityFor(role), domain.PermissionChange{
				RoleID:    fixture.roleIDs[domain.RoleEmployee],
				Module:    domain.ModuleAudit,
				Action:    domain.ActionRead,
				IsAllowed: true,
			}, RequestMeta{},
		)

		assert.ErrorIsf(t, err, domain.ErrForbidden, "role %s tidak boleh mengubah matriks", role)
		assert.Empty(t, fixture.permissions.upserted)
	}
}

// Pelanggaran invariant produk membatalkan transaction sehingga tidak ada side effect.
func TestUpdatePermissionRejectsInvariantViolationWithoutSideEffect(t *testing.T) {
	fixture := newAccessFixture()

	_, err := fixture.service.UpdatePermission(
		context.Background(), identityFor(domain.RoleHR), domain.PermissionChange{
			RoleID:    fixture.roleIDs[domain.RoleTopManagement],
			Module:    domain.ModuleAccess,
			Action:    domain.ActionUpdate,
			IsAllowed: true,
		}, RequestMeta{},
	)

	assert.ErrorIs(t, err, domain.ErrPermissionInvariant)
	assert.Empty(t, fixture.permissions.upserted)
	assert.Empty(t, fixture.audit.entries)
	assert.Empty(t, fixture.cache.invalidated)
}

func TestUpdatePermissionRejectsUnknownRole(t *testing.T) {
	fixture := newAccessFixture()

	_, err := fixture.service.UpdatePermission(
		context.Background(), identityFor(domain.RoleHR), domain.PermissionChange{
			RoleID:    uuid.New(),
			Module:    domain.ModuleAudit,
			Action:    domain.ActionRead,
			IsAllowed: true,
		}, RequestMeta{},
	)

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Empty(t, fixture.permissions.upserted)
}

// Kegagalan invalidasi cache harus membatalkan perubahan agar database dan cache tidak
// berbeda pendapat tentang kewenangan.
func TestUpdatePermissionFailsWhenCacheInvalidationFails(t *testing.T) {
	fixture := newAccessFixture()
	fixture.cache.err = errors.New("redis tidak tersedia")

	_, err := fixture.service.UpdatePermission(
		context.Background(), identityFor(domain.RoleHR), domain.PermissionChange{
			RoleID:    fixture.roleIDs[domain.RoleSupervisor],
			Module:    domain.ModuleOvertime,
			Action:    domain.ActionApprove,
			IsAllowed: true,
		}, RequestMeta{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "update permission")
}

func TestListAuditLogsRedactsSensitiveDetail(t *testing.T) {
	fixture := newAccessFixture()
	fixture.audits.page = domain.AuditLogPage{
		Page:  1,
		Limit: 10,
		Total: 1,
		Items: []domain.AuditLogEntry{{
			ID:     uuid.New(),
			Action: "UPDATE",
			Module: "karyawan",
			Detail: map[string]any{"password_hash": "rahasia", "modul": "karyawan"},
		}},
	}

	page, err := fixture.service.ListAuditLogs(
		context.Background(), identityFor(domain.RoleHR), domain.AuditLogFilter{},
	)

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, domain.AuditRedactedPlaceholder, page.Items[0].Detail["password_hash"])
	assert.Equal(t, "karyawan", page.Items[0].Detail["modul"])
}

func TestListAuditLogsClampsPaging(t *testing.T) {
	fixture := newAccessFixture()

	_, err := fixture.service.ListAuditLogs(
		context.Background(), identityFor(domain.RoleHR),
		domain.AuditLogFilter{Page: 0, Limit: 900},
	)

	require.NoError(t, err)
	assert.Equal(t, 1, fixture.audits.filter.Page)
	assert.Equal(t, 100, fixture.audits.filter.Limit)
}
