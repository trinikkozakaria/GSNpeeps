package tests

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type accessFixture struct {
	pool          *pgxpool.Pool
	notifications *repository.NotificationRepository
	permissions   *repository.PermissionRepository
	audits        *repository.AuditRepository
	tx            *repository.TransactionManager

	subordinate   uuid.UUID // user karyawan dengan atasan
	supervisor    uuid.UUID // user atasan langsung
	humanResource uuid.UUID // user HR aktif
	topManagement uuid.UUID // satu-satunya user Top Management aktif

	subordinateEmployee uuid.UUID
	employeeIDs         []uuid.UUID
	userIDs             []uuid.UUID
	departments         []uuid.UUID
	positions           []uuid.UUID
	contractIDs         []uuid.UUID
}

func newAccessFixture(t *testing.T) *accessFixture {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL tidak diset; integration test dilewati")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))

	suffix := uuid.NewString()[:8]
	f := &accessFixture{
		pool:          pool,
		notifications: repository.NewNotificationRepository(pool),
		permissions:   repository.NewPermissionRepository(pool),
		audits:        repository.NewAuditRepository(pool),
		tx:            repository.NewTransactionManager(pool),
	}

	var departmentID, positionID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO departments (nama) VALUES ($1) RETURNING id`, "Uji Akses "+suffix,
	).Scan(&departmentID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO positions (nama, department_id) VALUES ($1, $2) RETURNING id`,
		"Uji Posisi Akses "+suffix, departmentID,
	).Scan(&positionID))
	f.departments = append(f.departments, departmentID)
	f.positions = append(f.positions, positionID)

	supervisorEmployee := f.insertEmployee(t, "AKS-ATS-"+suffix, nil, departmentID, positionID)
	subordinateEmployee := f.insertEmployee(
		t, "AKS-KRY-"+suffix, &supervisorEmployee, departmentID, positionID,
	)
	humanResourceEmployee := f.insertEmployee(t, "AKS-HR-"+suffix, nil, departmentID, positionID)
	f.subordinateEmployee = subordinateEmployee

	f.supervisor = f.insertUser(t, supervisorEmployee, "aks-atasan-"+suffix+"@example.test", "atasan")
	f.subordinate = f.insertUser(t, subordinateEmployee, "aks-karyawan-"+suffix+"@example.test", "karyawan")
	f.humanResource = f.insertUser(t, humanResourceEmployee, "aks-hr-"+suffix+"@example.test", "hr")

	// Kontrak hanya boleh memiliki satu Top Management aktif; test memakai yang sudah ada bila
	// tersedia dan membuatnya bila belum.
	f.topManagement = f.resolveOrCreateTopManagement(t, suffix, departmentID, positionID)

	t.Cleanup(func() { f.cleanup() })
	return f
}

func (f *accessFixture) insertEmployee(
	t *testing.T, nip string, supervisor *uuid.UUID, departmentID, positionID uuid.UUID,
) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(context.Background(), `
		INSERT INTO employees (
			nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join,
			department_id, position_id, atasan_id, status
		)
		VALUES ($1, $2, 'L', '1994-02-02', '2026-01-05', $3, $4, $5, 'aktif')
		RETURNING id
	`, nip, "Pegawai "+nip, departmentID, positionID, supervisor).Scan(&id))
	f.employeeIDs = append(f.employeeIDs, id)
	return id
}

func (f *accessFixture) insertUser(
	t *testing.T, employeeID uuid.UUID, email, role string,
) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(context.Background(), `
		INSERT INTO users (employee_id, email, password_hash, role_id)
		VALUES ($1, $2, 'x', (SELECT id FROM roles WHERE nama = $3))
		RETURNING id
	`, employeeID, email, role).Scan(&id))
	f.userIDs = append(f.userIDs, id)
	return id
}

func (f *accessFixture) resolveOrCreateTopManagement(
	t *testing.T, suffix string, departmentID, positionID uuid.UUID,
) uuid.UUID {
	t.Helper()
	ids, err := f.notifications.ActiveUserIDsByRole(context.Background(), domain.RoleTopManagement)
	require.NoError(t, err)
	if len(ids) > 0 {
		return ids[0]
	}
	employeeID := f.insertEmployee(t, "AKS-TM-"+suffix, nil, departmentID, positionID)
	return f.insertUser(t, employeeID, "aks-tm-"+suffix+"@example.test", "top_management")
}

func (f *accessFixture) insertContract(t *testing.T, endDate string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(context.Background(), `
		INSERT INTO employee_contracts (
			employee_id, nomor_kontrak, tanggal_mulai, tanggal_berakhir, jenis_kontrak, status
		)
		VALUES ($1, $2, '2026-01-05', $3, 'PKWT', 'aktif')
		RETURNING id
	`, f.subordinateEmployee, "KTR-"+uuid.NewString()[:12], endDate).Scan(&id))
	f.contractIDs = append(f.contractIDs, id)
	return id
}

func (f *accessFixture) cleanup() {
	ctx := context.Background()
	_, _ = f.pool.Exec(ctx, `DELETE FROM notifications WHERE recipient_user_id = ANY($1)`, f.userIDs)
	_, _ = f.pool.Exec(ctx, `DELETE FROM employee_contracts WHERE id = ANY($1)`, f.contractIDs)

	// Audit Log append-only bagi aplikasi. Membersihkannya memerlukan hak DDL yang sengaja
	// tidak dimiliki service; test menonaktifkan trigger sementara agar user uji dapat dihapus.
	_, _ = f.pool.Exec(ctx, `ALTER TABLE audit_logs DISABLE TRIGGER trg_audit_logs_append_only`)
	_, _ = f.pool.Exec(ctx, `DELETE FROM audit_logs WHERE user_id = ANY($1)`, f.userIDs)
	_, _ = f.pool.Exec(ctx, `ALTER TABLE audit_logs ENABLE TRIGGER trg_audit_logs_append_only`)

	_, _ = f.pool.Exec(ctx, `DELETE FROM users WHERE id = ANY($1)`, f.userIDs)
	_, _ = f.pool.Exec(ctx, `UPDATE employees SET atasan_id = NULL WHERE id = ANY($1)`, f.employeeIDs)
	_, _ = f.pool.Exec(ctx, `DELETE FROM employees WHERE id = ANY($1)`, f.employeeIDs)
	_, _ = f.pool.Exec(ctx, `DELETE FROM positions WHERE id = ANY($1)`, f.positions)
	_, _ = f.pool.Exec(ctx, `DELETE FROM departments WHERE id = ANY($1)`, f.departments)
	f.pool.Close()
}

func (f *accessFixture) draft(recipient uuid.UUID, eventKey string) domain.NotificationDraft {
	reference := domain.ReferenceLeave
	referenceID := uuid.New()
	return domain.NotificationDraft{
		RecipientUserID: recipient,
		Type:            domain.NotificationLeaveSubmitted,
		Title:           "Pengajuan ketidakhadiran baru",
		Message:         "Ada pengajuan ketidakhadiran yang menunggu keputusan Anda.",
		ReferenceID:     &referenceID,
		ReferenceType:   &reference,
		EventKey:        eventKey,
		CreatedAt:       time.Now().UTC(),
	}
}

// Producer yang mengulang event yang sama hanya menghasilkan satu baris.
func TestNotificationInsertIsIdempotent(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	eventKey := "ketidakhadiran_baru:" + uuid.NewString() + ":atasan:" + f.supervisor.String()

	first, err := f.notifications.Insert(ctx, []domain.NotificationDraft{
		f.draft(f.supervisor, eventKey),
	})
	require.NoError(t, err)
	second, err := f.notifications.Insert(ctx, []domain.NotificationDraft{
		f.draft(f.supervisor, eventKey),
	})
	require.NoError(t, err)

	assert.Equal(t, 1, first)
	assert.Zero(t, second)
}

// Dua penulis bersamaan atas event yang sama diserialkan oleh UNIQUE constraint.
func TestConcurrentWritersCreateOneNotification(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	eventKey := "lembur_baru:" + uuid.NewString() + ":hr:" + f.humanResource.String()

	const writers = 6
	created := make([]int, writers)
	var group sync.WaitGroup
	group.Add(writers)
	for index := range writers {
		go func() {
			defer group.Done()
			count, err := f.notifications.Insert(ctx, []domain.NotificationDraft{
				f.draft(f.humanResource, eventKey),
			})
			if err == nil {
				created[index] = count
			}
		}()
	}
	group.Wait()

	total := 0
	for _, count := range created {
		total += count
	}
	assert.Equal(t, 1, total)
}

// Notifikasi yang sudah di-dismiss tidak boleh muncul kembali ketika producer retry.
func TestDismissedNotificationIsNeverRecreated(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	eventKey := "ketidakhadiran_baru:" + uuid.NewString() + ":hr:" + f.humanResource.String()

	_, err := f.notifications.Insert(ctx, []domain.NotificationDraft{
		f.draft(f.humanResource, eventKey),
	})
	require.NoError(t, err)

	page, err := f.notifications.List(ctx, f.humanResource, nil, 1, 10)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)

	require.NoError(t, f.notifications.Dismiss(ctx, f.humanResource, page.Items[0].ID, time.Now().UTC()))

	created, err := f.notifications.Insert(ctx, []domain.NotificationDraft{
		f.draft(f.humanResource, eventKey),
	})
	require.NoError(t, err)
	assert.Zero(t, created)

	afterRetry, err := f.notifications.List(ctx, f.humanResource, nil, 1, 10)
	require.NoError(t, err)
	assert.Empty(t, afterRetry.Items)

	// Baris tetap ada; dismiss adalah soft-delete, bukan penghapusan fisik.
	var dismissed *time.Time
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT dismissed_at FROM notifications WHERE id = $1`, page.Items[0].ID,
	).Scan(&dismissed))
	assert.NotNil(t, dismissed)
}

// Setiap operasi notifikasi terikat penerima; user lain tidak dapat membaca maupun mengubah.
func TestNotificationOperationsAreRowScoped(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	eventKey := "ketidakhadiran_baru:" + uuid.NewString() + ":atasan:" + f.supervisor.String()

	_, err := f.notifications.Insert(ctx, []domain.NotificationDraft{
		f.draft(f.supervisor, eventKey),
	})
	require.NoError(t, err)

	owned, err := f.notifications.List(ctx, f.supervisor, nil, 1, 10)
	require.NoError(t, err)
	require.Len(t, owned.Items, 1)
	notificationID := owned.Items[0].ID

	foreign, err := f.notifications.List(ctx, f.subordinate, nil, 1, 10)
	require.NoError(t, err)
	assert.Empty(t, foreign.Items)

	assert.ErrorIs(t,
		f.notifications.MarkRead(ctx, f.subordinate, notificationID, time.Now().UTC()),
		repository.ErrNotFound,
	)
	assert.ErrorIs(t,
		f.notifications.Dismiss(ctx, f.subordinate, notificationID, time.Now().UTC()),
		repository.ErrNotFound,
	)

	// Percobaan yang ditolak tidak meninggalkan efek apa pun.
	stillUnread, err := f.notifications.UnreadCount(ctx, f.supervisor)
	require.NoError(t, err)
	assert.Equal(t, 1, stillUnread)
}

func TestUnreadCountStaysCoherentWithListAndDismiss(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	reference := uuid.NewString()

	for _, stage := range []string{"atasan", "hr", "top_management"} {
		_, err := f.notifications.Insert(ctx, []domain.NotificationDraft{
			f.draft(f.subordinate, "ketidakhadiran_baru:"+reference+":"+stage+":"+f.subordinate.String()),
		})
		require.NoError(t, err)
	}

	unread, err := f.notifications.UnreadCount(ctx, f.subordinate)
	require.NoError(t, err)
	require.Equal(t, 3, unread)

	page, err := f.notifications.List(ctx, f.subordinate, nil, 1, 10)
	require.NoError(t, err)
	require.Len(t, page.Items, 3)

	require.NoError(t, f.notifications.MarkRead(ctx, f.subordinate, page.Items[0].ID, time.Now().UTC()))
	require.NoError(t, f.notifications.Dismiss(ctx, f.subordinate, page.Items[1].ID, time.Now().UTC()))

	unread, err = f.notifications.UnreadCount(ctx, f.subordinate)
	require.NoError(t, err)
	assert.Equal(t, 1, unread)

	visible, err := f.notifications.List(ctx, f.subordinate, nil, 1, 10)
	require.NoError(t, err)
	assert.Len(t, visible.Items, 2)

	isRead := false
	unreadOnly, err := f.notifications.List(ctx, f.subordinate, &isRead, 1, 10)
	require.NoError(t, err)
	assert.Len(t, unreadOnly.Items, 1)
}

// MarkRead berulang tetap berhasil dan tidak menggeser read_at yang sudah tercatat.
func TestMarkReadIsIdempotent(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	eventKey := "lembur_baru:" + uuid.NewString() + ":hr:" + f.humanResource.String()

	_, err := f.notifications.Insert(ctx, []domain.NotificationDraft{
		f.draft(f.humanResource, eventKey),
	})
	require.NoError(t, err)
	page, err := f.notifications.List(ctx, f.humanResource, nil, 1, 10)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)

	first := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, f.notifications.MarkRead(ctx, f.humanResource, page.Items[0].ID, first))
	require.NoError(t, f.notifications.MarkRead(
		ctx, f.humanResource, page.Items[0].ID, first.Add(time.Hour),
	))

	var readAt time.Time
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT read_at FROM notifications WHERE id = $1`, page.Items[0].ID,
	).Scan(&readAt))
	assert.WithinDuration(t, first, readAt, time.Millisecond)
}

// Resolusi penerima approver mengikuti relasi organisasi yang aktif.
func TestApproverRecipientsFollowOrganisationRelations(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()

	supervisors, err := f.notifications.ApproverUserIDs(ctx, domain.StageSupervisor, f.subordinate)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{f.supervisor}, supervisors)

	// Karyawan tanpa atasan tidak menghasilkan penerima tahap Atasan.
	none, err := f.notifications.ApproverUserIDs(ctx, domain.StageSupervisor, f.supervisor)
	require.NoError(t, err)
	assert.Empty(t, none)

	humanResources, err := f.notifications.ApproverUserIDs(ctx, domain.StageHR, f.subordinate)
	require.NoError(t, err)
	assert.Contains(t, humanResources, f.humanResource)

	topManagement, err := f.notifications.ApproverUserIDs(ctx, domain.StageTopManagement, f.humanResource)
	require.NoError(t, err)
	assert.Contains(t, topManagement, f.topManagement)
}

// Atasan yang sudah nonaktif tidak boleh menjadi penerima notifikasi.
func TestInactiveSupervisorIsNotAResolvedRecipient(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()

	_, err := f.pool.Exec(ctx, `
		UPDATE employees SET status = 'nonaktif', deleted_at = NOW()
		WHERE id = (SELECT employee_id FROM users WHERE id = $1)
	`, f.supervisor)
	require.NoError(t, err)

	recipients, err := f.notifications.ApproverUserIDs(ctx, domain.StageSupervisor, f.subordinate)
	require.NoError(t, err)
	assert.Empty(t, recipients)
}

// Job H-30 memindai kontrak berdasarkan tanggal berakhir yang tepat.
func TestExpiringContractsAreClaimedByExactEndDate(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	target := time.Date(2027, 3, 15, 0, 0, 0, 0, domain.Jakarta())
	contractID := f.insertContract(t, "2027-03-15")
	f.insertContract(t, "2027-03-16")

	claimed, err := f.notifications.ClaimExpiringContracts(ctx, target, 50)
	require.NoError(t, err)

	matched := []domain.ExpiringContract{}
	for _, candidate := range claimed {
		if candidate.ContractID == contractID {
			matched = append(matched, candidate)
		}
	}
	require.Len(t, matched, 1)
	assert.Equal(t, f.subordinateEmployee, matched[0].EmployeeID)
	assert.Equal(t, f.subordinate, matched[0].UserID)
	assert.Equal(t, "2027-03-15", matched[0].EndDate)
}

// Audit Log tidak dapat diubah maupun dihapus melalui koneksi aplikasi.
func TestAuditLogRejectsUpdateAndDelete(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	dataID := uuid.New()

	require.NoError(t, f.audits.Append(ctx, domain.AuditEntry{
		UserID:    &f.humanResource,
		Action:    "LOGIN",
		Module:    "auth",
		DataID:    &dataID,
		Detail:    map[string]any{"request_id": "req-audit"},
		IPAddress: "203.0.113.10",
		CreatedAt: time.Now().UTC(),
	}))

	_, err := f.pool.Exec(ctx,
		`UPDATE audit_logs SET aksi = 'TAMPERED' WHERE user_id = $1`, f.humanResource)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append-only")

	_, err = f.pool.Exec(ctx, `DELETE FROM audit_logs WHERE user_id = $1`, f.humanResource)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append-only")

	_, err = f.pool.Exec(ctx, `TRUNCATE audit_logs`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append-only")

	page, err := f.audits.List(ctx, domain.AuditLogFilter{
		UserID: &f.humanResource, Page: 1, Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "LOGIN", page.Items[0].Action)
}

func TestAuditLogFilterNarrowsByModuleActionAndDate(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()

	for _, entry := range []domain.AuditEntry{
		{UserID: &f.humanResource, Action: "CREATE", Module: "karyawan", CreatedAt: now},
		{UserID: &f.humanResource, Action: "APPROVE", Module: "ketidakhadiran", CreatedAt: now},
		{UserID: &f.supervisor, Action: "CREATE", Module: "karyawan", CreatedAt: now},
	} {
		require.NoError(t, f.audits.Append(ctx, entry))
	}

	module := "karyawan"
	byUserAndModule, err := f.audits.List(ctx, domain.AuditLogFilter{
		UserID: &f.humanResource, Module: &module, Page: 1, Limit: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, byUserAndModule.Total)

	action := "APPROVE"
	byAction, err := f.audits.List(ctx, domain.AuditLogFilter{
		UserID: &f.humanResource, Action: &action, Page: 1, Limit: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, byAction.Total)

	future := now.Add(48 * time.Hour)
	outsideRange, err := f.audits.List(ctx, domain.AuditLogFilter{
		UserID: &f.humanResource, StartDate: &future, Page: 1, Limit: 10,
	})
	require.NoError(t, err)
	assert.Zero(t, outsideRange.Total)
}

// Matriks permission memakai kosakata kontrak sehingga dapat dikembalikan dan diperbarui.
func TestPermissionMatrixUsesContractVocabulary(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()

	roles, err := f.permissions.ListRoles(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 4)
	for _, role := range roles {
		assert.Truef(t, role.Name.Valid(), "role %s tidak dikenal", role.Name)
		assert.NotEmpty(t, role.Description)
	}

	matrix, err := f.permissions.ListPermissions(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, matrix)
	for _, permission := range matrix {
		assert.Truef(t, domain.KnownCapability(permission.Module, permission.Action),
			"kapabilitas %s.%s di luar katalog kontrak", permission.Module, permission.Action)
	}
}

func TestUpsertPermissionUpdatesExistingRow(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()

	roles, err := f.permissions.ListRoles(ctx)
	require.NoError(t, err)
	var supervisorRole uuid.UUID
	for _, role := range roles {
		if role.Name == domain.RoleSupervisor {
			supervisorRole = role.ID
		}
	}
	require.NotEqual(t, uuid.Nil, supervisorRole)

	change := domain.PermissionChange{
		RoleID:    supervisorRole,
		Module:    domain.ModuleAttendanceReport,
		Action:    domain.ActionRead,
		IsAllowed: true,
	}
	before, err := f.permissions.CurrentPermission(ctx, change)
	require.NoError(t, err)
	t.Cleanup(func() {
		restore := change
		restore.IsAllowed = before
		_, _ = f.permissions.UpsertPermission(context.Background(), restore)
	})

	id, err := f.permissions.UpsertPermission(ctx, change)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)

	allowed, err := f.permissions.HasPermission(
		ctx, domain.RoleSupervisor, domain.ModuleAttendanceReport, domain.ActionRead,
	)
	require.NoError(t, err)
	assert.True(t, allowed)

	// Upsert kedua memperbarui baris yang sama, bukan menambah duplikat.
	change.IsAllowed = false
	sameID, err := f.permissions.UpsertPermission(ctx, change)
	require.NoError(t, err)
	assert.Equal(t, id, sameID)
}

func TestRolePermissionsReturnsCapabilityMap(t *testing.T) {
	f := newAccessFixture(t)

	capabilities, err := f.permissions.RolePermissions(context.Background(), domain.RoleHR)

	require.NoError(t, err)
	assert.True(t, capabilities[domain.ModuleAccess+"."+domain.ActionUpdate])
	assert.True(t, capabilities[domain.ModuleAudit+"."+domain.ActionRead])

	staff, err := f.permissions.RolePermissions(context.Background(), domain.RoleEmployee)
	require.NoError(t, err)
	assert.False(t, staff[domain.ModuleAccess+"."+domain.ActionRead])
	assert.False(t, staff[domain.ModuleAudit+"."+domain.ActionRead])
}

// Top Management read-only pada AKSES juga tercermin pada data seed.
func TestSeededMatrixKeepsTopManagementReadOnly(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()

	allowed, err := f.permissions.HasPermission(
		ctx, domain.RoleTopManagement, domain.ModuleAccess, domain.ActionUpdate,
	)
	require.NoError(t, err)
	assert.False(t, allowed)

	readable, err := f.permissions.HasPermission(
		ctx, domain.RoleTopManagement, domain.ModuleAccess, domain.ActionRead,
	)
	require.NoError(t, err)
	assert.True(t, readable)
}
