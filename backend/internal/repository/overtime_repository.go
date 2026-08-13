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

type OvertimeRepository struct {
	pool *pgxpool.Pool
}

func NewOvertimeRepository(pool *pgxpool.Pool) *OvertimeRepository {
	return &OvertimeRepository{pool: pool}
}

const overtimeJoins = `
	FROM overtime_requests o
	JOIN users u ON u.id = o.user_id
	JOIN roles pemohon_role ON pemohon_role.id = u.role_id
	JOIN employees e ON e.id = u.employee_id`

// CreateRequest menyimpan pengajuan lembur. Durasi tidak dikirim karena dihitung database
// sebagai generated column dari jam mulai dan selesai.
func (r *OvertimeRepository) CreateRequest(
	ctx context.Context,
	row domain.OvertimeRequestRow,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO overtime_requests (
			user_id, tanggal, jam_mulai, jam_selesai, alasan, dokumen_url, status
		)
		VALUES ($1, $2::date, $3::time, $4::time, $5, $6, $7)
		RETURNING id
	`, row.UserID, row.Date, row.StartTime, row.EndTime, row.Reason, row.DocumentURL, row.Status,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, mapEmployeeMutationError(err)
	}
	return id, nil
}

func overtimeScopeClause(scope domain.LeaveRequestScope, args *[]any) string {
	clauses := make([]string, 0, 4)
	if scope.RequesterUserID != nil {
		*args = append(*args, *scope.RequesterUserID)
		clauses = append(clauses, fmt.Sprintf("o.user_id = $%d", len(*args)))
	}
	if scope.SupervisorEmployeeID != nil {
		*args = append(*args, *scope.SupervisorEmployeeID)
		clauses = append(clauses, fmt.Sprintf("e.atasan_id = $%d", len(*args)))
	}
	if scope.Stage != nil {
		*args = append(*args, string(*scope.Stage))
		clauses = append(clauses, fmt.Sprintf("o.status = $%d", len(*args)))
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

func (r *OvertimeRepository) ListRequests(
	ctx context.Context,
	filter domain.OvertimeRequestFilter,
) (domain.OvertimeRequestPage, error) {
	args := make([]any, 0, 8)
	where := overtimeScopeClause(filter.Scope, &args)
	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		where += fmt.Sprintf(" AND o.status = $%d", len(args))
	}
	if filter.Start != nil {
		args = append(args, *filter.Start)
		where += fmt.Sprintf(" AND o.tanggal >= $%d::date", len(args))
	}
	if filter.End != nil {
		args = append(args, *filter.End)
		where += fmt.Sprintf(" AND o.tanggal <= $%d::date", len(args))
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*)`+overtimeJoins+` WHERE `+where, args...,
	).Scan(&total); err != nil {
		return domain.OvertimeRequestPage{}, fmt.Errorf("count overtime requests: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, e.id, e.nama, TO_CHAR(o.tanggal, 'YYYY-MM-DD'),
		       TO_CHAR(o.jam_mulai, 'HH24:MI:SS'), TO_CHAR(o.jam_selesai, 'HH24:MI:SS'),
		       o.durasi_jam::float8, o.status, o.created_at`+overtimeJoins+`
		WHERE `+where+`
		ORDER BY o.created_at DESC, o.id
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return domain.OvertimeRequestPage{}, fmt.Errorf("query overtime requests: %w", err)
	}
	defer rows.Close()

	items := make([]domain.OvertimeRequestSummary, 0)
	for rows.Next() {
		var item domain.OvertimeRequestSummary
		if err := rows.Scan(
			&item.ID, &item.EmployeeID, &item.EmployeeName, &item.Date,
			&item.StartTime, &item.EndTime, &item.TotalHours, &item.Status, &item.CreatedAt,
		); err != nil {
			return domain.OvertimeRequestPage{}, fmt.Errorf("scan overtime request: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.OvertimeRequestPage{}, fmt.Errorf("iterate overtime requests: %w", err)
	}
	return domain.OvertimeRequestPage{
		Items: items, Total: total, Page: filter.Page, Limit: filter.Limit,
	}, nil
}

func (r *OvertimeRepository) FindRequest(
	ctx context.Context,
	id uuid.UUID,
) (domain.OvertimeRequestDetail, error) {
	var detail domain.OvertimeRequestDetail
	err := r.pool.QueryRow(ctx, `
		SELECT o.id, e.id, e.nama, TO_CHAR(o.tanggal, 'YYYY-MM-DD'),
		       TO_CHAR(o.jam_mulai, 'HH24:MI:SS'), TO_CHAR(o.jam_selesai, 'HH24:MI:SS'),
		       o.durasi_jam::float8, o.status, o.created_at, o.alasan, o.dokumen_url`+overtimeJoins+`
		WHERE o.id = $1
	`, id).Scan(
		&detail.ID, &detail.EmployeeID, &detail.EmployeeName, &detail.Date,
		&detail.StartTime, &detail.EndTime, &detail.TotalHours, &detail.Status,
		&detail.CreatedAt, &detail.Reason, &detail.DocumentURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OvertimeRequestDetail{}, ErrNotFound
	}
	if err != nil {
		return domain.OvertimeRequestDetail{}, fmt.Errorf("find overtime request: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT oa.tahap, oa.approver_id, e.nama, oa.keputusan, oa.catatan, oa.created_at
		FROM overtime_approvals oa
		LEFT JOIN users u ON u.id = oa.approver_id
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE oa.overtime_request_id = $1
		ORDER BY oa.created_at, oa.id
	`, id)
	if err != nil {
		return domain.OvertimeRequestDetail{}, fmt.Errorf("query overtime approval history: %w", err)
	}
	defer rows.Close()

	history, err := scanApprovalHistory(rows)
	if err != nil {
		return domain.OvertimeRequestDetail{}, err
	}
	detail.ApprovalHistory = history
	return detail, nil
}

// LockRequestForDecision mengunci baris lembur untuk keputusan. Nilai kuota dan jumlah hari
// tidak relevan untuk lembur karena tidak ada saldo yang dikurangi.
func (r *OvertimeRepository) LockRequestForDecision(
	ctx context.Context,
	id uuid.UUID,
) (domain.RequestLock, error) {
	var lock domain.RequestLock
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT o.id, o.user_id, e.id, e.atasan_id, o.status
		FROM overtime_requests o
		JOIN users u ON u.id = o.user_id
		JOIN employees e ON e.id = u.employee_id
		WHERE o.id = $1
		FOR UPDATE OF o
	`, id).Scan(
		&lock.RequestID, &lock.RequesterUserID, &lock.RequesterEmployeeID,
		&lock.SupervisorEmployeeID, &lock.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RequestLock{}, ErrNotFound
	}
	if err != nil {
		return domain.RequestLock{}, fmt.Errorf("lock overtime request: %w", err)
	}
	return lock, nil
}

func (r *OvertimeRepository) UpdateRequestStatus(
	ctx context.Context,
	id uuid.UUID,
	from, to domain.RequestStatus,
) error {
	tag, err := executor(ctx, r.pool).Exec(ctx, `
		UPDATE overtime_requests
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND status = $2
	`, id, string(from), string(to))
	if err != nil {
		return fmt.Errorf("update overtime request status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

func (r *OvertimeRepository) AppendApproval(
	ctx context.Context,
	requestID uuid.UUID,
	stage domain.ApprovalStage,
	approverID *uuid.UUID,
	decision domain.ApprovalDecision,
	note *string,
) error {
	_, err := executor(ctx, r.pool).Exec(ctx, `
		INSERT INTO overtime_approvals (
			overtime_request_id, approver_id, tahap, keputusan, catatan
		)
		VALUES ($1, $2, $3, $4, $5)
	`, requestID, approverID, string(stage), string(decision), note)
	if err != nil {
		return fmt.Errorf("append overtime approval: %w", err)
	}
	return nil
}

// Recap menghitung total pengajuan dan jam lembur yang disetujui per karyawan. Kompensasi
// tidak dihitung; PRD menetapkan perhitungan dilakukan manual di luar sistem.
func (r *OvertimeRepository) Recap(
	ctx context.Context,
	filter domain.OvertimeRecapFilter,
) ([]domain.OvertimeRecapItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.nama, COALESCE(d.nama, ''),
		       COUNT(o.id), COALESCE(SUM(o.durasi_jam), 0)::float8
		FROM overtime_requests o
		JOIN users u ON u.id = o.user_id
		JOIN employees e ON e.id = u.employee_id
		LEFT JOIN departments d ON d.id = e.department_id
		WHERE o.status = 'disetujui'
		  AND ($1::date IS NULL OR o.tanggal >= $1::date)
		  AND ($2::date IS NULL OR o.tanggal <= $2::date)
		  AND ($3::uuid IS NULL OR e.department_id = $3)
		GROUP BY e.id, e.nama, d.nama
		ORDER BY e.nama, e.id
	`, filter.Start, filter.End, filter.DepartmentID)
	if err != nil {
		return nil, fmt.Errorf("query overtime recap: %w", err)
	}
	defer rows.Close()

	items := make([]domain.OvertimeRecapItem, 0)
	for rows.Next() {
		var item domain.OvertimeRecapItem
		if err := rows.Scan(
			&item.EmployeeID, &item.EmployeeName, &item.Department,
			&item.TotalRequest, &item.TotalHours,
		); err != nil {
			return nil, fmt.Errorf("scan overtime recap: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *OvertimeRepository) ClaimEscalatableRequests(
	ctx context.Context,
	before time.Time,
	limit int,
) ([]domain.EscalationCandidate, error) {
	rows, err := executor(ctx, r.pool).Query(ctx, `
		SELECT id, user_id
		FROM overtime_requests
		WHERE status = 'menunggu_atasan' AND created_at < $1
		ORDER BY created_at
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, before, limit)
	if err != nil {
		return nil, fmt.Errorf("claim escalatable overtime requests: %w", err)
	}
	defer rows.Close()

	items := make([]domain.EscalationCandidate, 0)
	for rows.Next() {
		var item domain.EscalationCandidate
		if err := rows.Scan(&item.RequestID, &item.RequesterUserID); err != nil {
			return nil, fmt.Errorf("scan escalatable overtime request: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// LeaveMetrics memenuhi kontrak metrik dashboard: hari izin final disetujui dan jumlah
// pengajuan yang masih menunggu keputusan pada modul ketidakhadiran maupun lembur.
func (r *OvertimeRepository) LeaveMetrics(
	ctx context.Context,
	period domain.DashboardRange,
) (domain.LeaveMetrics, error) {
	start := period.Start.Format(domain.DateLayout)
	end := period.End.Format(domain.DateLayout)

	var metrics domain.LeaveMetrics
	// Hari izin dihitung sebagai irisan hari Senin-Jumat antara rentang izin dan periode.
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(
			(SELECT COUNT(*)
			 FROM generate_series(
			     GREATEST(lr.tanggal_mulai, $1::date),
			     LEAST(lr.tanggal_selesai, $2::date),
			     INTERVAL '1 day'
			 ) AS hari
			 WHERE EXTRACT(ISODOW FROM hari) < 6)
		), 0)
		FROM leave_requests lr
		WHERE lr.status = 'disetujui'
		  AND lr.tanggal_mulai <= $2::date AND lr.tanggal_selesai >= $1::date
	`, start, end).Scan(&metrics.ApprovedLeaveDays)
	if err != nil {
		return domain.LeaveMetrics{}, fmt.Errorf("aggregate approved leave days: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		SELECT (
			SELECT COUNT(*) FROM leave_requests
			WHERE status IN ('menunggu_atasan', 'menunggu_hr', 'menunggu_top_management')
		) + (
			SELECT COUNT(*) FROM overtime_requests
			WHERE status IN ('menunggu_atasan', 'menunggu_hr', 'menunggu_top_management')
		)
	`).Scan(&metrics.PendingRequests)
	if err != nil {
		return domain.LeaveMetrics{}, fmt.Errorf("aggregate pending requests: %w", err)
	}
	return metrics, nil
}
