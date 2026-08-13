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

type approvalFixture struct {
	pool        *pgxpool.Pool
	leaves      *repository.LeaveRepository
	overtimes   *repository.OvertimeRepository
	tx          *repository.TransactionManager
	leaveTypeID uuid.UUID
	requester   uuid.UUID // user id pemohon
	approver    uuid.UUID // user id atasan
	employeeIDs []uuid.UUID
	departments []uuid.UUID
	positions   []uuid.UUID
}

func newApprovalFixture(t *testing.T) *approvalFixture {
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
	f := &approvalFixture{
		pool:      pool,
		leaves:    repository.NewLeaveRepository(pool),
		overtimes: repository.NewOvertimeRepository(pool),
		tx:        repository.NewTransactionManager(pool),
	}

	var departmentID, positionID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO departments (nama) VALUES ($1) RETURNING id`, "Uji Approval "+suffix,
	).Scan(&departmentID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO positions (nama, department_id) VALUES ($1, $2) RETURNING id`,
		"Uji Posisi "+suffix, departmentID,
	).Scan(&positionID))
	f.departments = append(f.departments, departmentID)
	f.positions = append(f.positions, positionID)

	approverEmployee := f.insertEmployee(t, "APV-"+suffix, "Atasan Sintetis", nil, departmentID, positionID)
	requesterEmployee := f.insertEmployee(
		t, "REQ-"+suffix, "Pemohon Sintetis", &approverEmployee, departmentID, positionID,
	)
	f.approver = f.insertUser(t, approverEmployee, "atasan-"+suffix+"@example.test", "atasan")
	f.requester = f.insertUser(t, requesterEmployee, "pemohon-"+suffix+"@example.test", "karyawan")

	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO leave_types (kode, nama, kuota_tahunan, memerlukan_dokumen)
		VALUES ($1, $2, 12, FALSE) RETURNING id
	`, "UJI-"+suffix, "Uji Cuti "+suffix).Scan(&f.leaveTypeID))

	_, err = pool.Exec(ctx, `
		INSERT INTO leave_balances (user_id, tahun, saldo_awal) VALUES ($1, 2026, 12)
	`, f.requester)
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanup := context.Background()
		_, _ = pool.Exec(cleanup, `DELETE FROM leave_approvals WHERE leave_request_id IN (
			SELECT id FROM leave_requests WHERE user_id = ANY($1))`, []uuid.UUID{f.requester, f.approver})
		_, _ = pool.Exec(cleanup, `DELETE FROM overtime_approvals WHERE overtime_request_id IN (
			SELECT id FROM overtime_requests WHERE user_id = ANY($1))`, []uuid.UUID{f.requester, f.approver})
		_, _ = pool.Exec(cleanup, `DELETE FROM leave_requests WHERE user_id = ANY($1)`,
			[]uuid.UUID{f.requester, f.approver})
		_, _ = pool.Exec(cleanup, `DELETE FROM overtime_requests WHERE user_id = ANY($1)`,
			[]uuid.UUID{f.requester, f.approver})
		_, _ = pool.Exec(cleanup, `DELETE FROM leave_balances WHERE user_id = ANY($1)`,
			[]uuid.UUID{f.requester, f.approver})
		// Audit Log append-only bagi aplikasi; trigger dinonaktifkan sementara agar user uji
		// yang direferensikan audit tetap dapat dihapus.
		_, _ = pool.Exec(cleanup, `ALTER TABLE audit_logs DISABLE TRIGGER trg_audit_logs_append_only`)
		_, _ = pool.Exec(cleanup, `DELETE FROM audit_logs WHERE user_id = ANY($1)`,
			[]uuid.UUID{f.requester, f.approver})
		_, _ = pool.Exec(cleanup, `ALTER TABLE audit_logs ENABLE TRIGGER trg_audit_logs_append_only`)
		_, _ = pool.Exec(cleanup, `DELETE FROM leave_types WHERE id = $1`, f.leaveTypeID)
		_, _ = pool.Exec(cleanup, `DELETE FROM users WHERE id = ANY($1)`,
			[]uuid.UUID{f.requester, f.approver})
		_, _ = pool.Exec(cleanup, `UPDATE employees SET atasan_id = NULL WHERE id = ANY($1)`, f.employeeIDs)
		_, _ = pool.Exec(cleanup, `DELETE FROM employees WHERE id = ANY($1)`, f.employeeIDs)
		_, _ = pool.Exec(cleanup, `DELETE FROM positions WHERE id = ANY($1)`, f.positions)
		_, _ = pool.Exec(cleanup, `DELETE FROM departments WHERE id = ANY($1)`, f.departments)
		pool.Close()
	})
	return f
}

func (f *approvalFixture) insertEmployee(
	t *testing.T, nip, name string, supervisor *uuid.UUID, departmentID, positionID uuid.UUID,
) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(context.Background(), `
		INSERT INTO employees (
			nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join,
			department_id, position_id, atasan_id, status
		)
		VALUES ($1, $2, 'P', '1995-01-01', '2026-01-05', $3, $4, $5, 'aktif')
		RETURNING id
	`, nip, name, departmentID, positionID, supervisor).Scan(&id))
	f.employeeIDs = append(f.employeeIDs, id)
	return id
}

func (f *approvalFixture) insertUser(
	t *testing.T, employeeID uuid.UUID, email, role string,
) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(context.Background(), `
		INSERT INTO users (employee_id, email, password_hash, role_id)
		VALUES ($1, $2, 'x', (SELECT id FROM roles WHERE nama = $3))
		RETURNING id
	`, employeeID, email, role).Scan(&id))
	return id
}

func (f *approvalFixture) createLeaveRequest(
	t *testing.T, status domain.RequestStatus, createdAt *time.Time,
) uuid.UUID {
	t.Helper()
	id, err := f.leaves.CreateRequest(context.Background(), domain.LeaveRequestRow{
		UserID:      f.requester,
		LeaveTypeID: f.leaveTypeID,
		StartDate:   "2026-08-10",
		EndDate:     "2026-08-12",
		TotalDays:   3,
		Reason:      "Keperluan keluarga sintetis",
		Status:      status,
	})
	require.NoError(t, err)
	if createdAt != nil {
		_, err = f.pool.Exec(context.Background(),
			`UPDATE leave_requests SET created_at = $2 WHERE id = $1`, id, *createdAt)
		require.NoError(t, err)
	}
	return id
}

func (f *approvalFixture) statusOf(t *testing.T, id uuid.UUID) domain.RequestStatus {
	t.Helper()
	var status domain.RequestStatus
	require.NoError(t, f.pool.QueryRow(context.Background(),
		`SELECT status FROM leave_requests WHERE id = $1`, id).Scan(&status))
	return status
}

// Dua keputusan bersaing harus menghasilkan tepat satu pemenang.
func TestConcurrentDecisionsProduceSingleWinner(t *testing.T) {
	f := newApprovalFixture(t)
	requestID := f.createLeaveRequest(t, domain.StatusWaitingSupervisor, nil)

	decide := func(to domain.RequestStatus) error {
		return f.tx.Within(context.Background(), func(ctx context.Context) error {
			if _, err := f.leaves.LockRequestForDecision(ctx, requestID); err != nil {
				return err
			}
			// Jeda memperbesar peluang kedua transaksi tumpang tindih.
			time.Sleep(50 * time.Millisecond)
			return f.leaves.UpdateRequestStatus(
				ctx, requestID, domain.StatusWaitingSupervisor, to,
			)
		})
	}

	var wg sync.WaitGroup
	results := make([]error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); results[0] = decide(domain.StatusWaitingHR) }()
	go func() { defer wg.Done(); results[1] = decide(domain.StatusRejected) }()
	wg.Wait()

	succeeded, conflicted := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			succeeded++
		case err == repository.ErrConflict:
			conflicted++
		default:
			t.Fatalf("error tak terduga: %v", err)
		}
	}
	assert.Equal(t, 1, succeeded, "hanya satu keputusan yang boleh menang")
	assert.Equal(t, 1, conflicted, "keputusan yang kalah harus menerima conflict")
	assert.NotEqual(t, domain.StatusWaitingSupervisor, f.statusOf(t, requestID))
}

// Delegasi yang kalah bersaing dengan keputusan harus gagal, bukan menimpa status.
func TestDecisionVersusDelegateProducesSingleWinner(t *testing.T) {
	f := newApprovalFixture(t)
	requestID := f.createLeaveRequest(t, domain.StatusWaitingSupervisor, nil)
	ctx := context.Background()

	require.NoError(t, f.leaves.UpdateRequestStatus(
		ctx, requestID, domain.StatusWaitingSupervisor, domain.StatusRejected,
	))
	// Delegasi setelah pengajuan ditolak harus gagal karena status sudah berubah.
	err := f.leaves.UpdateRequestStatus(
		ctx, requestID, domain.StatusWaitingSupervisor, domain.StatusWaitingHR,
	)
	assert.ErrorIs(t, err, repository.ErrConflict)
	assert.Equal(t, domain.StatusRejected, f.statusOf(t, requestID))
}

// Saldo hanya boleh berkurang sekali dan tidak boleh menjadi negatif.
func TestLeaveBalanceDeductsOnceAndNeverGoesNegative(t *testing.T) {
	f := newApprovalFixture(t)
	ctx := context.Background()

	require.NoError(t, f.leaves.DeductLeaveBalance(ctx, f.requester, 2026, 10))
	balance, err := f.leaves.FindLeaveBalance(ctx, f.requester, 2026)
	require.NoError(t, err)
	assert.Equal(t, 10, balance.Used)
	assert.Equal(t, 2, balance.Remaining)

	// Pengurangan berikutnya melebihi sisa saldo dan harus ditolak tanpa mengubah data.
	err = f.leaves.DeductLeaveBalance(ctx, f.requester, 2026, 5)
	assert.ErrorIs(t, err, repository.ErrConflict)

	balance, err = f.leaves.FindLeaveBalance(ctx, f.requester, 2026)
	require.NoError(t, err)
	assert.Equal(t, 10, balance.Used, "saldo tidak boleh berubah setelah pengurangan gagal")
	assert.Equal(t, 2, balance.Remaining)
}

// Worker eskalasi hanya mengambil pengajuan tahap atasan yang melewati SLA 2x24 jam.
func TestEscalationClaimsOnlyRequestsPastSLA(t *testing.T) {
	f := newApprovalFixture(t)
	ctx := context.Background()

	stale := time.Now().UTC().Add(-domain.EscalationThreshold - time.Hour)
	fresh := time.Now().UTC().Add(-time.Hour)
	staleID := f.createLeaveRequest(t, domain.StatusWaitingSupervisor, &stale)
	freshID := f.createLeaveRequest(t, domain.StatusWaitingSupervisor, &fresh)
	// Pengajuan tahap HR tidak boleh tereskalasi; tidak ada SLA HR ke Top Management.
	hrStale := stale
	hrID := f.createLeaveRequest(t, domain.StatusWaitingHR, &hrStale)

	threshold := time.Now().UTC().Add(-domain.EscalationThreshold)
	var claimed []domain.EscalationCandidate
	require.NoError(t, f.tx.Within(ctx, func(txContext context.Context) error {
		var err error
		claimed, err = f.leaves.ClaimEscalatableRequests(txContext, threshold, 100)
		return err
	}))

	ids := make(map[uuid.UUID]bool, len(claimed))
	for _, candidate := range claimed {
		ids[candidate.RequestID] = true
	}
	assert.True(t, ids[staleID], "pengajuan melewati SLA harus diklaim")
	assert.False(t, ids[freshID], "pengajuan belum melewati SLA tidak boleh diklaim")
	assert.False(t, ids[hrID], "tahap HR tidak memiliki auto-escalation")
}

// Auto-escalation menyimpan riwayat tanpa approver dan aman dijalankan ulang.
func TestEscalationIsIdempotentAcrossRuns(t *testing.T) {
	f := newApprovalFixture(t)
	ctx := context.Background()
	stale := time.Now().UTC().Add(-domain.EscalationThreshold - time.Hour)
	requestID := f.createLeaveRequest(t, domain.StatusWaitingSupervisor, &stale)

	escalate := func() error {
		return f.tx.Within(ctx, func(txContext context.Context) error {
			if err := f.leaves.UpdateRequestStatus(
				txContext, requestID, domain.StatusWaitingSupervisor, domain.StatusWaitingHR,
			); err != nil {
				return err
			}
			return f.leaves.AppendApproval(
				txContext, requestID, domain.StageSupervisor, nil,
				domain.DecisionAutoEscalate, nil,
			)
		})
	}

	require.NoError(t, escalate())
	// Eksekusi kedua tidak menemukan pengajuan pada tahap atasan sehingga tidak berdampak.
	assert.ErrorIs(t, escalate(), repository.ErrConflict)

	assert.Equal(t, domain.StatusWaitingHR, f.statusOf(t, requestID))

	var historyCount int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM leave_approvals WHERE leave_request_id = $1`, requestID,
	).Scan(&historyCount))
	assert.Equal(t, 1, historyCount, "riwayat eskalasi tidak boleh terduplikasi")

	detail, err := f.leaves.FindRequest(ctx, requestID)
	require.NoError(t, err)
	require.Len(t, detail.ApprovalHistory, 1)
	assert.Equal(t, "auto_eskalasi", detail.ApprovalHistory[0].Decision)
	assert.Nil(t, detail.ApprovalHistory[0].ApproverID, "keputusan sistem tidak memiliki approver")
}

// Constraint database menolak riwayat auto_escalate yang membawa approver.
func TestAutoEscalateHistoryRejectsApprover(t *testing.T) {
	f := newApprovalFixture(t)
	requestID := f.createLeaveRequest(t, domain.StatusWaitingSupervisor, nil)

	err := f.leaves.AppendApproval(
		context.Background(), requestID, domain.StageSupervisor, &f.approver,
		domain.DecisionAutoEscalate, nil,
	)
	assert.Error(t, err, "auto_escalate harus memiliki approver NULL")
}

// Durasi lembur dihitung database dari jam mulai dan selesai.
func TestOvertimeDurationIsComputedByDatabase(t *testing.T) {
	f := newApprovalFixture(t)
	ctx := context.Background()

	id, err := f.overtimes.CreateRequest(ctx, domain.OvertimeRequestRow{
		UserID:    f.requester,
		Date:      "2026-08-10",
		StartTime: "18:00:00",
		EndTime:   "20:30:00",
		Reason:    "Penyelesaian rilis sintetis",
		Status:    domain.StatusWaitingSupervisor,
	})
	require.NoError(t, err)

	detail, err := f.overtimes.FindRequest(ctx, id)
	require.NoError(t, err)
	assert.InDelta(t, 2.5, detail.TotalHours, 0.001)
	assert.Nil(t, detail.DocumentURL, "dokumen lembur bersifat opsional")
}

// Jam selesai harus setelah jam mulai; database menegakkannya lewat CHECK.
func TestOvertimeRejectsEndBeforeStart(t *testing.T) {
	f := newApprovalFixture(t)

	_, err := f.overtimes.CreateRequest(context.Background(), domain.OvertimeRequestRow{
		UserID:    f.requester,
		Date:      "2026-08-10",
		StartTime: "20:00:00",
		EndTime:   "18:00:00",
		Reason:    "Rentang tidak valid",
		Status:    domain.StatusWaitingSupervisor,
	})
	assert.Error(t, err)
}
