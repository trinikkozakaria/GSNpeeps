package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeaveRepository struct {
	pool *pgxpool.Pool
}

func NewLeaveRepository(pool *pgxpool.Pool) *LeaveRepository {
	return &LeaveRepository{pool: pool}
}

func (r *LeaveRepository) ListLeaveTypes(
	ctx context.Context,
	activeOnly *bool,
) ([]domain.LeaveType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, kode, nama, kuota_tahunan, memerlukan_dokumen, is_active
		FROM leave_types
		WHERE $1::boolean IS NULL OR is_active = $1
		ORDER BY nama, id
	`, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("list leave types: %w", err)
	}
	defer rows.Close()

	items := make([]domain.LeaveType, 0)
	for rows.Next() {
		var item domain.LeaveType
		if err := rows.Scan(
			&item.ID, &item.Code, &item.Name, &item.AnnualQuota,
			&item.RequiresDocument, &item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan leave type: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *LeaveRepository) FindLeaveType(
	ctx context.Context,
	id uuid.UUID,
) (domain.LeaveType, error) {
	var item domain.LeaveType
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT id, kode, nama, kuota_tahunan, memerlukan_dokumen, is_active
		FROM leave_types WHERE id = $1
	`, id).Scan(
		&item.ID, &item.Code, &item.Name, &item.AnnualQuota,
		&item.RequiresDocument, &item.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LeaveType{}, ErrNotFound
	}
	if err != nil {
		return domain.LeaveType{}, fmt.Errorf("find leave type: %w", err)
	}
	return item, nil
}

func (r *LeaveRepository) CreateLeaveType(
	ctx context.Context,
	command domain.CreateLeaveType,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO leave_types (kode, nama, kuota_tahunan, memerlukan_dokumen)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, command.Code, command.Name, command.AnnualQuota, command.RequiresDocument).Scan(&id)
	if err != nil {
		return uuid.Nil, mapEmployeeMutationError(err)
	}
	return id, nil
}

func (r *LeaveRepository) UpdateLeaveType(
	ctx context.Context,
	id uuid.UUID,
	changes domain.UpdateLeaveType,
) error {
	setParts := make([]string, 0, 4)
	args := []any{id}
	add := func(column string, value any) {
		args = append(args, value)
		setParts = append(setParts, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if changes.Name != nil {
		add("nama", *changes.Name)
	}
	if changes.AnnualQuota != nil {
		add("kuota_tahunan", *changes.AnnualQuota)
	}
	if changes.RequiresDocument != nil {
		add("memerlukan_dokumen", *changes.RequiresDocument)
	}
	if changes.IsActive != nil {
		add("is_active", *changes.IsActive)
	}
	if len(setParts) == 0 {
		return nil
	}
	setParts = append(setParts, "updated_at = NOW()")

	tag, err := executor(ctx, r.pool).Exec(ctx, `
		UPDATE leave_types SET `+strings.Join(setParts, ", ")+` WHERE id = $1
	`, args...)
	if err != nil {
		return mapEmployeeMutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateRequest menyimpan pengajuan ketidakhadiran beserta status awal hasil routing.
func (r *LeaveRepository) CreateRequest(
	ctx context.Context,
	row domain.LeaveRequestRow,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO leave_requests (
			user_id, leave_type_id, tanggal_mulai, tanggal_selesai, jumlah_hari,
			alasan, dokumen_url, lokasi_tujuan, keperluan_tugas, status
		)
		VALUES ($1, $2, $3::date, $4::date, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`,
		row.UserID, row.LeaveTypeID, row.StartDate, row.EndDate, row.TotalDays,
		row.Reason, row.DocumentURL, row.Destination, row.DestinationNote, row.Status,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, mapEmployeeMutationError(err)
	}
	return id, nil
}

// leaveScopeClause membangun predikat row-level dari scope yang ditetapkan service.
func leaveScopeClause(scope domain.LeaveRequestScope, args *[]any) string {
	clauses := make([]string, 0, 4)
	if scope.RequesterUserID != nil {
		*args = append(*args, *scope.RequesterUserID)
		clauses = append(clauses, fmt.Sprintf("lr.user_id = $%d", len(*args)))
	}
	if scope.SupervisorEmployeeID != nil {
		*args = append(*args, *scope.SupervisorEmployeeID)
		clauses = append(clauses, fmt.Sprintf("e.atasan_id = $%d", len(*args)))
	}
	if scope.Stage != nil {
		*args = append(*args, string(*scope.Stage))
		clauses = append(clauses, fmt.Sprintf("lr.status = $%d", len(*args)))
	}
	if scope.RequesterRole != nil {
		*args = append(*args, string(*scope.RequesterRole))
		clauses = append(clauses, fmt.Sprintf("pemohon_role.nama = $%d", len(*args)))
	}
	if len(clauses) == 0 {
		return "TRUE"
	}
	return strings.Join(clauses, " AND ")
}

const leaveJoins = `
	FROM leave_requests lr
	JOIN users u ON u.id = lr.user_id
	JOIN roles pemohon_role ON pemohon_role.id = u.role_id
	JOIN employees e ON e.id = u.employee_id
	JOIN leave_types lt ON lt.id = lr.leave_type_id`

func (r *LeaveRepository) ListRequests(
	ctx context.Context,
	filter domain.LeaveRequestFilter,
) (domain.LeaveRequestPage, error) {
	args := make([]any, 0, 6)
	where := leaveScopeClause(filter.Scope, &args)
	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		where += fmt.Sprintf(" AND lr.status = $%d", len(args))
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*)`+leaveJoins+` WHERE `+where, args...,
	).Scan(&total); err != nil {
		return domain.LeaveRequestPage{}, fmt.Errorf("count leave requests: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, `
		SELECT lr.id, e.id, e.nama, lt.nama,
		       TO_CHAR(lr.tanggal_mulai, 'YYYY-MM-DD'),
		       TO_CHAR(lr.tanggal_selesai, 'YYYY-MM-DD'),
		       lr.jumlah_hari, lr.status, lr.created_at`+leaveJoins+`
		WHERE `+where+`
		ORDER BY lr.created_at DESC, lr.id
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return domain.LeaveRequestPage{}, fmt.Errorf("query leave requests: %w", err)
	}
	defer rows.Close()

	items := make([]domain.LeaveRequestSummary, 0)
	for rows.Next() {
		var item domain.LeaveRequestSummary
		if err := rows.Scan(
			&item.ID, &item.EmployeeID, &item.EmployeeName, &item.LeaveType,
			&item.StartDate, &item.EndDate, &item.TotalDays, &item.Status, &item.CreatedAt,
		); err != nil {
			return domain.LeaveRequestPage{}, fmt.Errorf("scan leave request: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.LeaveRequestPage{}, fmt.Errorf("iterate leave requests: %w", err)
	}
	return domain.LeaveRequestPage{
		Items: items, Total: total, Page: filter.Page, Limit: filter.Limit,
	}, nil
}

// FindRequest mengembalikan detail tanpa menerapkan scope; pemanggil wajib memeriksa hak
// akses memakai RequestAccess sebelum mengirimkan hasilnya.
func (r *LeaveRepository) FindRequest(
	ctx context.Context,
	id uuid.UUID,
) (domain.LeaveRequestDetail, error) {
	var detail domain.LeaveRequestDetail
	err := r.pool.QueryRow(ctx, `
		SELECT lr.id, e.id, e.nama, lt.nama,
		       TO_CHAR(lr.tanggal_mulai, 'YYYY-MM-DD'),
		       TO_CHAR(lr.tanggal_selesai, 'YYYY-MM-DD'),
		       lr.jumlah_hari, lr.status, lr.created_at,
		       lr.alasan, lr.dokumen_url, lr.lokasi_tujuan, lr.keperluan_tugas`+leaveJoins+`
		WHERE lr.id = $1
	`, id).Scan(
		&detail.ID, &detail.EmployeeID, &detail.EmployeeName, &detail.LeaveType,
		&detail.StartDate, &detail.EndDate, &detail.TotalDays, &detail.Status,
		&detail.CreatedAt, &detail.Reason, &detail.DocumentURL,
		&detail.Destination, &detail.DestinationNote,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LeaveRequestDetail{}, ErrNotFound
	}
	if err != nil {
		return domain.LeaveRequestDetail{}, fmt.Errorf("find leave request: %w", err)
	}

	history, err := r.approvalHistory(ctx, id)
	if err != nil {
		return domain.LeaveRequestDetail{}, err
	}
	detail.ApprovalHistory = history
	return detail, nil
}

func (r *LeaveRepository) approvalHistory(
	ctx context.Context,
	requestID uuid.UUID,
) ([]domain.ApprovalHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT la.tahap, la.approver_id, e.nama, la.keputusan, la.catatan, la.created_at
		FROM leave_approvals la
		LEFT JOIN users u ON u.id = la.approver_id
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE la.leave_request_id = $1
		ORDER BY la.created_at, la.id
	`, requestID)
	if err != nil {
		return nil, fmt.Errorf("query leave approval history: %w", err)
	}
	defer rows.Close()

	return scanApprovalHistory(rows)
}

// scanApprovalHistory memetakan baris riwayat approval menjadi bentuk response. Nilai
// keputusan database diterjemahkan ke enum kontrak oleh domain.NewApprovalHistory (D-025).
func scanApprovalHistory(rows pgx.Rows) ([]domain.ApprovalHistory, error) {
	items := make([]domain.ApprovalHistory, 0)
	for rows.Next() {
		var (
			stage      domain.ApprovalStage
			approverID *uuid.UUID
			name       *string
			decision   domain.ApprovalDecision
			note       *string
			decidedAt  time.Time
		)
		if err := rows.Scan(&stage, &approverID, &name, &decision, &note, &decidedAt); err != nil {
			return nil, fmt.Errorf("scan approval history: %w", err)
		}
		items = append(items, domain.NewApprovalHistory(
			stage, approverID, name, decision, note, decidedAt,
		))
	}
	return items, rows.Err()
}

// LockRequestForDecision mengunci baris pengajuan dan mengembalikan state yang dibutuhkan
// untuk memvalidasi keputusan. FOR UPDATE membuat dua keputusan bersaing diserialkan:
// pemenang mengubah status, pihak yang kalah membaca status baru dan gagal dengan
// ErrAlreadyDecided pada lapisan service.
func (r *LeaveRepository) LockRequestForDecision(
	ctx context.Context,
	id uuid.UUID,
) (domain.RequestLock, error) {
	var lock domain.RequestLock
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT lr.id, lr.user_id, e.id, e.atasan_id, lr.status, lr.jumlah_hari,
		       lt.kuota_tahunan, lt.id, EXTRACT(YEAR FROM lr.tanggal_mulai)::int
		FROM leave_requests lr
		JOIN users u ON u.id = lr.user_id
		JOIN employees e ON e.id = u.employee_id
		JOIN leave_types lt ON lt.id = lr.leave_type_id
		WHERE lr.id = $1
		FOR UPDATE OF lr
	`, id).Scan(
		&lock.RequestID, &lock.RequesterUserID, &lock.RequesterEmployeeID,
		&lock.SupervisorEmployeeID, &lock.Status, &lock.TotalDays,
		&lock.AnnualQuota, &lock.LeaveTypeID, &lock.Year,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RequestLock{}, ErrNotFound
	}
	if err != nil {
		return domain.RequestLock{}, fmt.Errorf("lock leave request: %w", err)
	}
	return lock, nil
}

// UpdateRequestStatus menerapkan perubahan status secara kondisional. RowsAffected nol
// berarti status sudah berubah sejak dibaca sehingga keputusan ini kalah bersaing.
func (r *LeaveRepository) UpdateRequestStatus(
	ctx context.Context,
	id uuid.UUID,
	from, to domain.RequestStatus,
) error {
	tag, err := executor(ctx, r.pool).Exec(ctx, `
		UPDATE leave_requests
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND status = $2
	`, id, string(from), string(to))
	if err != nil {
		return fmt.Errorf("update leave request status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// AppendApproval menyimpan satu baris riwayat keputusan.
func (r *LeaveRepository) AppendApproval(
	ctx context.Context,
	requestID uuid.UUID,
	stage domain.ApprovalStage,
	approverID *uuid.UUID,
	decision domain.ApprovalDecision,
	note *string,
) error {
	_, err := executor(ctx, r.pool).Exec(ctx, `
		INSERT INTO leave_approvals (leave_request_id, approver_id, tahap, keputusan, catatan)
		VALUES ($1, $2, $3, $4, $5)
	`, requestID, approverID, string(stage), string(decision), note)
	if err != nil {
		return fmt.Errorf("append leave approval: %w", err)
	}
	return nil
}

// DeductLeaveBalance mengurangi saldo secara atomic. Constraint saldo_terpakai <= saldo_awal
// menolak saldo negatif di level database, dan predikat pada WHERE mencegah pengurangan
// ganda ketika dua transaksi mencoba menyelesaikan pengajuan yang sama.
func (r *LeaveRepository) DeductLeaveBalance(
	ctx context.Context,
	userID uuid.UUID,
	year int,
	days int,
) error {
	tag, err := executor(ctx, r.pool).Exec(ctx, `
		UPDATE leave_balances
		SET saldo_terpakai = saldo_terpakai + $3, updated_at = NOW()
		WHERE user_id = $1 AND tahun = $2 AND saldo_awal - saldo_terpakai >= $3
	`, userID, year, days)
	if err != nil {
		return fmt.Errorf("deduct leave balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// FindLeaveBalance membaca saldo satu user pada satu tahun.
func (r *LeaveRepository) FindLeaveBalance(
	ctx context.Context,
	userID uuid.UUID,
	year int,
) (domain.LeaveBalance, error) {
	var balance domain.LeaveBalance
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT user_id, tahun, saldo_awal, saldo_terpakai, saldo_sisa
		FROM leave_balances WHERE user_id = $1 AND tahun = $2
	`, userID, year).Scan(
		&balance.UserID, &balance.Year, &balance.Opening, &balance.Used, &balance.Remaining,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LeaveBalance{}, ErrNotFound
	}
	if err != nil {
		return domain.LeaveBalance{}, fmt.Errorf("find leave balance: %w", err)
	}
	return balance, nil
}

// ClaimEscalatableRequests mengambil pengajuan tahap Atasan yang melewati SLA. SKIP LOCKED
// membuat beberapa worker dapat berjalan bersamaan tanpa memproses baris yang sama.
func (r *LeaveRepository) ClaimEscalatableRequests(
	ctx context.Context,
	before time.Time,
	limit int,
) ([]domain.EscalationCandidate, error) {
	rows, err := executor(ctx, r.pool).Query(ctx, `
		SELECT id, user_id
		FROM leave_requests
		WHERE status = 'menunggu_atasan' AND created_at < $1
		ORDER BY created_at
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, before, limit)
	if err != nil {
		return nil, fmt.Errorf("claim escalatable leave requests: %w", err)
	}
	defer rows.Close()

	items := make([]domain.EscalationCandidate, 0)
	for rows.Next() {
		var item domain.EscalationCandidate
		if err := rows.Scan(&item.RequestID, &item.RequesterUserID); err != nil {
			return nil, fmt.Errorf("scan escalatable leave request: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
