package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentApprovalRepository struct {
	pool *pgxpool.Pool
}

func NewDocumentApprovalRepository(pool *pgxpool.Pool) *DocumentApprovalRepository {
	return &DocumentApprovalRepository{pool: pool}
}

// --- Workflow templates -----------------------------------------------------------

func (r *DocumentApprovalRepository) ListTemplates(
	ctx context.Context,
	activeOnly *bool,
) ([]domain.WorkflowTemplate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, nama, aktif, created_at, updated_at
		FROM document_workflow_templates
		WHERE $1::boolean IS NULL OR aktif = $1
		ORDER BY nama, id
	`, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("list workflow templates: %w", err)
	}
	defer rows.Close()

	items := make([]domain.WorkflowTemplate, 0)
	for rows.Next() {
		var item domain.WorkflowTemplate
		if err := rows.Scan(&item.ID, &item.Name, &item.Active, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workflow template: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *DocumentApprovalRepository) FindTemplate(
	ctx context.Context,
	id uuid.UUID,
) (domain.WorkflowTemplate, []domain.WorkflowStage, error) {
	var item domain.WorkflowTemplate
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT id, nama, aktif, created_at, updated_at
		FROM document_workflow_templates WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.Active, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WorkflowTemplate{}, nil, ErrNotFound
	}
	if err != nil {
		return domain.WorkflowTemplate{}, nil, fmt.Errorf("find workflow template: %w", err)
	}

	stages, err := r.listStages(ctx, id)
	if err != nil {
		return domain.WorkflowTemplate{}, nil, err
	}
	return item, stages, nil
}

func (r *DocumentApprovalRepository) listStages(
	ctx context.Context,
	templateID uuid.UUID,
) ([]domain.WorkflowStage, error) {
	rows, err := executor(ctx, r.pool).Query(ctx, `
		SELECT id, template_id, urutan, nama_tahap, position_id
		FROM document_workflow_stages
		WHERE template_id = $1
		ORDER BY urutan
	`, templateID)
	if err != nil {
		return nil, fmt.Errorf("list workflow stages: %w", err)
	}
	defer rows.Close()

	items := make([]domain.WorkflowStage, 0)
	for rows.Next() {
		var item domain.WorkflowStage
		if err := rows.Scan(&item.ID, &item.TemplateID, &item.Order, &item.StageName, &item.PositionID); err != nil {
			return nil, fmt.Errorf("scan workflow stage: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CreateTemplate inserts the template row plus every stage in one statement batch. The
// caller (service layer) is expected to run this inside s.tx.Within so a failure partway
// through leaves nothing behind.
func (r *DocumentApprovalRepository) CreateTemplate(
	ctx context.Context,
	command domain.CreateWorkflowTemplate,
) (uuid.UUID, error) {
	var templateID uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO document_workflow_templates (nama, aktif)
		VALUES ($1, $2)
		RETURNING id
	`, command.Name, command.Active).Scan(&templateID)
	if err != nil {
		return uuid.Nil, mapEmployeeMutationError(err)
	}

	for _, stage := range command.Stages {
		if _, err := executor(ctx, r.pool).Exec(ctx, `
			INSERT INTO document_workflow_stages (template_id, urutan, nama_tahap, position_id)
			VALUES ($1, $2, $3, $4)
		`, templateID, stage.Order, stage.StageName, stage.PositionID); err != nil {
			return uuid.Nil, mapEmployeeMutationError(err)
		}
	}
	return templateID, nil
}

// UpdateTemplate replaces the stage list wholesale when changes.Stages is non-nil —
// simpler and less error-prone than diffing add/remove/reorder against existing rows,
// and matches how the form page (§9.7) always submits the full stage array.
func (r *DocumentApprovalRepository) UpdateTemplate(
	ctx context.Context,
	id uuid.UUID,
	changes domain.UpdateWorkflowTemplate,
) error {
	if changes.Name != nil || changes.Active != nil {
		setParts := make([]string, 0, 2)
		args := []any{id}
		add := func(column string, value any) {
			args = append(args, value)
			setParts = append(setParts, fmt.Sprintf("%s = $%d", column, len(args)))
		}
		if changes.Name != nil {
			add("nama", *changes.Name)
		}
		if changes.Active != nil {
			add("aktif", *changes.Active)
		}
		setParts = append(setParts, "updated_at = NOW()")
		tag, err := executor(ctx, r.pool).Exec(ctx, `
			UPDATE document_workflow_templates SET `+strings.Join(setParts, ", ")+` WHERE id = $1
		`, args...)
		if err != nil {
			return mapEmployeeMutationError(err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
	}

	if changes.Stages != nil {
		if _, err := executor(ctx, r.pool).Exec(ctx, `
			DELETE FROM document_workflow_stages WHERE template_id = $1
		`, id); err != nil {
			return mapEmployeeMutationError(err)
		}
		for _, stage := range *changes.Stages {
			if _, err := executor(ctx, r.pool).Exec(ctx, `
				INSERT INTO document_workflow_stages (template_id, urutan, nama_tahap, position_id)
				VALUES ($1, $2, $3, $4)
			`, id, stage.Order, stage.StageName, stage.PositionID); err != nil {
				return mapEmployeeMutationError(err)
			}
		}
	}
	return nil
}

func (r *DocumentApprovalRepository) StageCount(ctx context.Context, templateID uuid.UUID) (int, error) {
	var count int
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT COUNT(*) FROM document_workflow_stages WHERE template_id = $1
	`, templateID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count workflow stages: %w", err)
	}
	return count, nil
}

func (r *DocumentApprovalRepository) StageAtOrder(
	ctx context.Context,
	templateID uuid.UUID,
	order int,
) (domain.WorkflowStage, error) {
	var item domain.WorkflowStage
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT id, template_id, urutan, nama_tahap, position_id
		FROM document_workflow_stages WHERE template_id = $1 AND urutan = $2
	`, templateID, order).Scan(&item.ID, &item.TemplateID, &item.Order, &item.StageName, &item.PositionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WorkflowStage{}, ErrNotFound
	}
	if err != nil {
		return domain.WorkflowStage{}, fmt.Errorf("find workflow stage: %w", err)
	}
	return item, nil
}

// --- Approval requests -------------------------------------------------------------

// FindActiveByNormalizedURL backs the idempotency check (§5.2). Returning ErrNotFound
// when nothing is running means "safe to create a new request."
func (r *DocumentApprovalRepository) FindActiveByNormalizedURL(
	ctx context.Context,
	normalizedURL string,
) (domain.ApprovalRequest, error) {
	var item domain.ApprovalRequest
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT id, template_id, document_url, document_url_normalized, judul,
		       pemohon_user_id, stage_index, status, created_at, updated_at
		FROM document_approval_requests
		WHERE document_url_normalized = $1 AND status = $2
	`, normalizedURL, string(domain.DocApprovalRunning)).Scan(
		&item.ID, &item.TemplateID, &item.DocumentURL, &item.DocumentURLNormalized, &item.Title,
		&item.RequesterUserID, &item.StageIndex, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ApprovalRequest{}, ErrNotFound
	}
	if err != nil {
		return domain.ApprovalRequest{}, fmt.Errorf("find active approval request: %w", err)
	}
	return item, nil
}

// CreateRequest relies on the partial unique index UNIQUE(document_url_normalized)
// WHERE status='berjalan' as the real race-safe guard — FindActiveByNormalizedURL above
// is only a pre-check for returning a friendly idempotent response; if two requests for
// the same URL race past that check simultaneously, this INSERT is what actually
// prevents the duplicate, and mapEmployeeMutationError below must translate the
// resulting unique-violation into ErrConflict for the service layer to catch.
func (r *DocumentApprovalRepository) CreateRequest(
	ctx context.Context,
	command domain.CreateApprovalRequest,
	normalizedURL string,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO document_approval_requests (
			template_id, document_url, document_url_normalized, judul,
			pemohon_user_id, stage_index, status
		)
		VALUES ($1, $2, $3, $4, $5, 0, $6)
		RETURNING id
	`, command.TemplateID, command.DocumentURL, normalizedURL, command.Title,
		command.RequesterUserID, string(domain.DocApprovalRunning),
	).Scan(&id)
	if err != nil {
		return uuid.Nil, mapEmployeeMutationError(err)
	}
	return id, nil
}

func (r *DocumentApprovalRepository) ListRequests(
	ctx context.Context,
	filter domain.ApprovalRequestFilter,
) (domain.ApprovalRequestPage, error) {
	args := make([]any, 0, 4)
	where := "TRUE"
	if filter.Scope.RequesterUserID != nil {
		args = append(args, *filter.Scope.RequesterUserID)
		where += fmt.Sprintf(" AND pemohon_user_id = $%d", len(args))
	}
	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if filter.TemplateID != nil {
		args = append(args, *filter.TemplateID)
		where += fmt.Sprintf(" AND template_id = $%d", len(args))
	}
	// Note: ApproverUserID scope is deliberately NOT applied here — resolving "is this
	// user an approver at the request's current stage" requires a live join against
	// employees/positions that depends on each row's own stage_index, which cannot be
	// expressed as a single flat WHERE without joining document_workflow_stages per
	// row. The service layer applies that filter in-memory after fetching a candidate
	// page. See DocumentApprovalService.ListForApproval.

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM document_approval_requests WHERE `+where, args...,
	).Scan(&total); err != nil {
		return domain.ApprovalRequestPage{}, fmt.Errorf("count approval requests: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, `
		SELECT id, template_id, document_url, document_url_normalized, judul,
		       pemohon_user_id, stage_index, status, created_at, updated_at
		FROM document_approval_requests
		WHERE `+where+`
		ORDER BY created_at DESC, id
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return domain.ApprovalRequestPage{}, fmt.Errorf("query approval requests: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ApprovalRequest, 0)
	for rows.Next() {
		var item domain.ApprovalRequest
		if err := rows.Scan(
			&item.ID, &item.TemplateID, &item.DocumentURL, &item.DocumentURLNormalized, &item.Title,
			&item.RequesterUserID, &item.StageIndex, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return domain.ApprovalRequestPage{}, fmt.Errorf("scan approval request: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.ApprovalRequestPage{}, fmt.Errorf("iterate approval requests: %w", err)
	}
	return domain.ApprovalRequestPage{Items: items, Total: total, Page: filter.Page, Limit: filter.Limit}, nil
}

func (r *DocumentApprovalRepository) FindRequest(
	ctx context.Context,
	id uuid.UUID,
) (domain.ApprovalRequest, error) {
	var item domain.ApprovalRequest
	err := r.pool.QueryRow(ctx, `
		SELECT id, template_id, document_url, document_url_normalized, judul,
		       pemohon_user_id, stage_index, status, created_at, updated_at
		FROM document_approval_requests WHERE id = $1
	`, id).Scan(
		&item.ID, &item.TemplateID, &item.DocumentURL, &item.DocumentURLNormalized, &item.Title,
		&item.RequesterUserID, &item.StageIndex, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ApprovalRequest{}, ErrNotFound
	}
	if err != nil {
		return domain.ApprovalRequest{}, fmt.Errorf("find approval request: %w", err)
	}
	return item, nil
}

func (r *DocumentApprovalRepository) DecisionHistory(
	ctx context.Context,
	requestID uuid.UUID,
) ([]domain.ApprovalDecisionRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.urutan_tahap, d.approver_user_id, e.nama, d.keputusan, d.catatan, d.decided_at
		FROM document_approval_decisions d
		JOIN users u ON u.id = d.approver_user_id
		JOIN employees e ON e.id = u.employee_id
		WHERE d.request_id = $1
		ORDER BY d.decided_at, d.id
	`, requestID)
	if err != nil {
		return nil, fmt.Errorf("query approval decision history: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ApprovalDecisionRecord, 0)
	for rows.Next() {
		var item domain.ApprovalDecisionRecord
		if err := rows.Scan(
			&item.StageOrder, &item.ApproverUserID, &item.ApproverName, &item.Decision, &item.Note, &item.DecidedAt,
		); err != nil {
			return nil, fmt.Errorf("scan approval decision: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *DocumentApprovalRepository) DecisionCount(ctx context.Context, requestID uuid.UUID) (int, error) {
	var count int
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT COUNT(*) FROM document_approval_decisions WHERE request_id = $1
	`, requestID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count approval decisions: %w", err)
	}
	return count, nil
}

// LockRequestForDecision — same FOR UPDATE pattern as leave. TotalStages is fetched in the
// same query so the service layer can determine "is this the final stage" without a
// second round trip.
func (r *DocumentApprovalRepository) LockRequestForDecision(
	ctx context.Context,
	id uuid.UUID,
) (domain.ApprovalRequestLock, error) {
	var lock domain.ApprovalRequestLock
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT lr.id, lr.template_id, lr.pemohon_user_id, lr.stage_index, lr.status,
		       (SELECT COUNT(*) FROM document_workflow_stages s WHERE s.template_id = lr.template_id)
		FROM document_approval_requests lr
		WHERE lr.id = $1
		FOR UPDATE OF lr
	`, id).Scan(
		&lock.RequestID, &lock.TemplateID, &lock.RequesterUserID, &lock.StageIndex, &lock.Status, &lock.TotalStages,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ApprovalRequestLock{}, ErrNotFound
	}
	if err != nil {
		return domain.ApprovalRequestLock{}, fmt.Errorf("lock approval request: %w", err)
	}
	return lock, nil
}

// AdvanceStage moves stage_index forward and conditionally checks both id AND the
// previous stage_index AND status, so a decision that raced against another one (e.g. two
// approvers on the same stage clicking simultaneously) fails safely with RowsAffected==0
// even though leave's simpler WHERE id+status alone would not catch a same-status,
// different-stage race — not possible in leave's model, but is here since status stays
// 'berjalan' across every non-final stage transition.
func (r *DocumentApprovalRepository) AdvanceStage(
	ctx context.Context,
	id uuid.UUID,
	fromStageIndex int,
	toStageIndex int,
) error {
	tag, err := executor(ctx, r.pool).Exec(ctx, `
		UPDATE document_approval_requests
		SET stage_index = $3, status = $4, updated_at = NOW()
		WHERE id = $1 AND stage_index = $2 AND status = $5
	`, id, fromStageIndex, toStageIndex, string(domain.DocApprovalRunning), string(domain.DocApprovalRunning))
	if err != nil {
		return fmt.Errorf("advance approval stage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// FinalizeStatus applies a terminal status (disetujui/ditolak/dibatalkan) or the
// 'bermasalah' blocked state, conditionally on the current stage_index+status still
// matching what the caller observed under the lock.
func (r *DocumentApprovalRepository) FinalizeStatus(
	ctx context.Context,
	id uuid.UUID,
	atStageIndex int,
	fromStatus domain.DocumentApprovalStatus,
	toStatus domain.DocumentApprovalStatus,
) error {
	tag, err := executor(ctx, r.pool).Exec(ctx, `
		UPDATE document_approval_requests
		SET status = $4, updated_at = NOW()
		WHERE id = $1 AND stage_index = $2 AND status = $3
	`, id, atStageIndex, string(fromStatus), string(toStatus))
	if err != nil {
		return fmt.Errorf("finalize approval status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

func (r *DocumentApprovalRepository) AppendDecision(
	ctx context.Context,
	requestID uuid.UUID,
	stageOrder int,
	approverUserID uuid.UUID,
	decision domain.DocumentDecision,
	note *string,
) error {
	_, err := executor(ctx, r.pool).Exec(ctx, `
		INSERT INTO document_approval_decisions (request_id, urutan_tahap, approver_user_id, keputusan, catatan)
		VALUES ($1, $2, $3, $4, $5)
	`, requestID, stageOrder, approverUserID, string(decision), note)
	if err != nil {
		return fmt.Errorf("append approval decision: %w", err)
	}
	return nil
}

// ResolveStageApprovers is the live lookup from PRD §5.1/§7: every active employee
// currently holding the stage's Position. Deliberately not cached/snapshotted — a
// Position's holder can change between initiation and decision, and the PRD requires
// live resolution at both points.
func (r *DocumentApprovalRepository) ResolveStageApprovers(
	ctx context.Context,
	positionID uuid.UUID,
) ([]domain.StageApprover, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, e.nama
		FROM employees e
		JOIN users u ON u.employee_id = e.id
		WHERE e.position_id = $1 AND e.status = 'aktif' AND e.deleted_at IS NULL
		ORDER BY e.nama
	`, positionID)
	if err != nil {
		return nil, fmt.Errorf("resolve stage approvers: %w", err)
	}
	defer rows.Close()

	items := make([]domain.StageApprover, 0)
	for rows.Next() {
		var item domain.StageApprover
		if err := rows.Scan(&item.UserID, &item.EmployeeName); err != nil {
			return nil, fmt.Errorf("scan stage approver: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// IsActiveHolderOfPosition is the fast, single-row check used to authorize a decision —
// cheaper than fetching the whole ResolveStageApprovers list when the service layer only
// needs a yes/no for one specific user.
func (r *DocumentApprovalRepository) IsActiveHolderOfPosition(
	ctx context.Context,
	userID uuid.UUID,
	positionID uuid.UUID,
) (bool, error) {
	var exists bool
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM employees e
			JOIN users u ON u.employee_id = e.id
			WHERE u.id = $1 AND e.position_id = $2 AND e.status = 'aktif' AND e.deleted_at IS NULL
		)
	`, userID, positionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check active position holder: %w", err)
	}
	return exists, nil
}