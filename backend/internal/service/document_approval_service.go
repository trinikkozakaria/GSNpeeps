package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

// DocumentApprovalStore is the storage surface this service needs. Kept as its own
// interface (rather than reusing LeaveStore) since the two modules share almost no
// method shapes once idempotency and live position-resolution enter the picture.
type DocumentApprovalStore interface {
	ListTemplates(context.Context, *bool) ([]domain.WorkflowTemplate, error)
	FindTemplate(context.Context, uuid.UUID) (domain.WorkflowTemplate, []domain.WorkflowStage, error)
	CreateTemplate(context.Context, domain.CreateWorkflowTemplate) (uuid.UUID, error)
	UpdateTemplate(context.Context, uuid.UUID, domain.UpdateWorkflowTemplate) error
	StageCount(context.Context, uuid.UUID) (int, error)
	StageAtOrder(context.Context, uuid.UUID, int) (domain.WorkflowStage, error)

	FindActiveByNormalizedURL(context.Context, string) (domain.ApprovalRequest, error)
	CreateRequest(context.Context, domain.CreateApprovalRequest, string) (uuid.UUID, error)
	ListRequests(context.Context, domain.ApprovalRequestFilter) (domain.ApprovalRequestPage, error)
	FindRequest(context.Context, uuid.UUID) (domain.ApprovalRequest, error)
	DecisionHistory(context.Context, uuid.UUID) ([]domain.ApprovalDecisionRecord, error)
	DecisionCount(context.Context, uuid.UUID) (int, error)

	LockRequestForDecision(context.Context, uuid.UUID) (domain.ApprovalRequestLock, error)
	AdvanceStage(ctx context.Context, id uuid.UUID, fromStageIndex, toStageIndex int) error
	FinalizeStatus(ctx context.Context, id uuid.UUID, atStageIndex int, from, to domain.DocumentApprovalStatus) error
	AppendDecision(context.Context, uuid.UUID, int, uuid.UUID, domain.DocumentDecision, *string) error

	ResolveStageApprovers(context.Context, uuid.UUID) ([]domain.StageApprover, error)
	IsActiveHolderOfPosition(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

type DocumentApprovalService struct {
	store DocumentApprovalStore
	tx    EmployeeTransactionManager
	audit AuditWriter
	now   func() time.Time
}

func NewDocumentApprovalService(
	store DocumentApprovalStore,
	tx EmployeeTransactionManager,
	audit AuditWriter,
) *DocumentApprovalService {
	return &DocumentApprovalService{store: store, tx: tx, audit: audit, now: time.Now}
}

// --- Template management (HR only) --------------------------------------------------

func (s *DocumentApprovalService) ListTemplates(
	ctx context.Context,
	identity domain.Identity,
) ([]domain.WorkflowTemplate, error) {
	activeOnly := (*bool)(nil)
	if identity.Role != domain.RoleHR {
		forced := true
		activeOnly = &forced
	}
	items, err := s.store.ListTemplates(ctx, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("list workflow templates: %w", err)
	}
	return items, nil
}

func (s *DocumentApprovalService) CreateTemplate(
	ctx context.Context,
	identity domain.Identity,
	command domain.CreateWorkflowTemplate,
	meta RequestMeta,
) (uuid.UUID, error) {
	if identity.Role != domain.RoleHR {
		return uuid.Nil, domain.ErrForbidden
	}
	command.Name = strings.TrimSpace(command.Name)
	if command.Name == "" || len(command.Stages) == 0 {
		return uuid.Nil, domain.ErrInvalidRequest
	}
	for i, stage := range command.Stages {
		if strings.TrimSpace(stage.StageName) == "" || stage.PositionID == uuid.Nil {
			return uuid.Nil, domain.ErrInvalidRequest
		}
		// Order is derived from array position on the frontend (PRD §9.7), but
		// re-validated here rather than trusted, since the request body is untrusted
		// input regardless of what the UI normally sends.
		if stage.Order != i {
			return uuid.Nil, domain.ErrInvalidRequest
		}
	}

	var id uuid.UUID
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		created, err := s.store.CreateTemplate(txContext, command)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		id = created
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "CREATE",
			Module: "master_alur_persetujuan_dokumen",
			DataID: &created,
			Detail: map[string]any{
				"nama":         command.Name,
				"jumlah_tahap": len(command.Stages),
				"request_id":   meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create workflow template: %w", err)
	}
	return id, nil
}

func (s *DocumentApprovalService) UpdateTemplate(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	changes domain.UpdateWorkflowTemplate,
	meta RequestMeta,
) error {
	if identity.Role != domain.RoleHR {
		return domain.ErrForbidden
	}
	if changes.Name == nil && changes.Active == nil && changes.Stages == nil {
		return domain.ErrInvalidRequest
	}
	if changes.Stages != nil {
		if len(*changes.Stages) == 0 {
			return domain.ErrInvalidRequest
		}
		for i, stage := range *changes.Stages {
			if strings.TrimSpace(stage.StageName) == "" || stage.PositionID == uuid.Nil || stage.Order != i {
				return domain.ErrInvalidRequest
			}
		}
	}

	err := s.tx.Within(ctx, func(txContext context.Context) error {
		if err := s.store.UpdateTemplate(txContext, id, changes); err != nil {
			return mapEmployeeRepositoryError(err)
		}
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "UPDATE",
			Module:    "master_alur_persetujuan_dokumen",
			DataID:    &id,
			Detail:    map[string]any{"request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return fmt.Errorf("update workflow template: %w", err)
	}
	return nil
}

// --- Submission -----------------------------------------------------------------------

// Create implements §6.1. Unlike leave's Create, this has no status-routing decision to
// make (every request always starts at stage 0), but it does have the idempotency check
// leave doesn't need, and a "no active holder at stage 1" branch that produces the
// 'bermasalah' blocked state instead of failing outright.
func (s *DocumentApprovalService) Create(
	ctx context.Context,
	identity domain.Identity,
	command domain.CreateApprovalRequest,
	meta RequestMeta,
) (domain.ApprovalRequest, bool, error) {
	// bool return = "was this an idempotent hit on an already-running request" — the
	// handler uses this to decide whether to return the freshly created request or the
	// pre-existing one, both as a 200 (PRD §5.2: idempotent hit is not an error).
	command.Title = strings.TrimSpace(command.Title)
	if command.Title == "" {
		return domain.ApprovalRequest{}, false, domain.ErrInvalidRequest
	}
	normalizedURL, err := domain.NormalizeDocumentURL(command.DocumentURL)
	if err != nil {
		return domain.ApprovalRequest{}, false, domain.ErrInvalidRequest
	}
	// Non-Goal per PRD §2: no HTTP reachability check against the URL — format
	// validation only, performed inside NormalizeDocumentURL.

	template, stages, err := s.store.FindTemplate(ctx, command.TemplateID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.ApprovalRequest{}, false, domain.ErrInvalidRequest
	}
	if err != nil {
		return domain.ApprovalRequest{}, false, fmt.Errorf("resolve workflow template: %w", err)
	}
	if !template.Active || len(stages) == 0 {
		return domain.ApprovalRequest{}, false, domain.ErrInvalidRequest
	}

	// Idempotency pre-check (§5.2). This is a convenience fast path for the common case;
	// the partial unique index in the schema is what actually prevents a race between two
	// simultaneous submissions for the same URL — see CreateRequest's doc comment.
	if existing, err := s.store.FindActiveByNormalizedURL(ctx, normalizedURL); err == nil {
		return existing, true, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return domain.ApprovalRequest{}, false, fmt.Errorf("check existing approval request: %w", err)
	}

	command.RequesterUserID = identity.UserID
	command.DocumentURL = strings.TrimSpace(command.DocumentURL)

	var result domain.ApprovalRequest
	err = s.tx.Within(ctx, func(txContext context.Context) error {
		requestID, err := s.store.CreateRequest(txContext, command, normalizedURL)
		if err != nil {
			if errors.Is(err, repository.ErrConflict) {
				// Lost the race against a concurrent identical submission — treat it the
				// same as the pre-check hit, not as an error.
				existing, findErr := s.store.FindActiveByNormalizedURL(txContext, normalizedURL)
				if findErr != nil {
					return findErr
				}
				result = existing
				return nil
			}
			return mapEmployeeRepositoryError(err)
		}

		firstStage := stages[0]
		approvers, err := s.store.ResolveStageApprovers(txContext, firstStage.PositionID)
		if err != nil {
			return fmt.Errorf("resolve first stage approvers: %w", err)
		}

		status := domain.DocApprovalRunning
		if len(approvers) == 0 {
			// PRD §13: no active holder at the active stage -> blocked, not failed.
			// Falls back to notifying HR rather than silently stalling.
			status = domain.DocApprovalBlocked
			if err := s.store.FinalizeStatus(
				txContext, requestID, 0, domain.DocApprovalRunning, domain.DocApprovalBlocked,
			); err != nil {
				return err
			}
		}

		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "INISIASI",
			Module: "persetujuan_dokumen",
			DataID: &requestID,
			Detail: map[string]any{
				"template_id": command.TemplateID,
				"status":      string(status),
				"request_id":  meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		}); err != nil {
			return err
		}

		// NOTE: notification dispatch (approvers = resolved holders of the active
		// stage) intentionally deferred — see notification follow-up task once
		// domain.NotificationDraft's exact shape is confirmed.

		result = domain.ApprovalRequest{
			ID: requestID, TemplateID: command.TemplateID, DocumentURL: command.DocumentURL,
			DocumentURLNormalized: normalizedURL, Title: command.Title,
			RequesterUserID: identity.UserID, StageIndex: 0, Status: status,
		}
		return nil
	})
	if err != nil {
		return domain.ApprovalRequest{}, false, fmt.Errorf("create approval request: %w", err)
	}
	return result, false, nil
}

// --- Reading ---------------------------------------------------------------------------

func (s *DocumentApprovalService) ListMine(
	ctx context.Context,
	identity domain.Identity,
	status *domain.DocumentApprovalStatus,
	page, limit int,
) (domain.ApprovalRequestPage, error) {
	page, limit = normalizePaging(page, limit)
	userID := identity.UserID
	result, err := s.store.ListRequests(ctx, domain.ApprovalRequestFilter{
		Scope:  domain.ApprovalRequestScope{RequesterUserID: &userID},
		Status: status, Page: page, Limit: limit,
	})
	if err != nil {
		return domain.ApprovalRequestPage{}, fmt.Errorf("list own approval requests: %w", err)
	}
	return result, nil
}

// ListForApproval backs the Inbox page (§9.3). Unlike leave's inbox, whether a request
// belongs in this user's inbox depends on live Position-holder resolution at the request's
// *current* stage — not a single flat column the repository can filter on directly. So
// this fetches a candidate page (running requests, since only 'berjalan' requests have an
// active decidable stage) and filters in-memory. Acceptable at this data scale (internal
// HR tool, not a public high-volume API); if this ever becomes a bottleneck, the fix is a
// materialized "current stage position_id" column kept in sync by AdvanceStage/FinalizeStatus,
// not a bigger in-memory filter.
func (s *DocumentApprovalService) ListForApproval(
	ctx context.Context,
	identity domain.Identity,
	page, limit int,
) (domain.ApprovalRequestPage, error) {
	page, limit = normalizePaging(page, limit)
	running := domain.DocApprovalRunning
	candidates, err := s.store.ListRequests(ctx, domain.ApprovalRequestFilter{
		Status: &running, Page: 1, Limit: 500, // candidate window; see doc comment above
	})
	if err != nil {
		return domain.ApprovalRequestPage{}, fmt.Errorf("list approval candidates: %w", err)
	}

	matched := make([]domain.ApprovalRequest, 0)
	for _, request := range candidates.Items {
		stage, err := s.store.StageAtOrder(ctx, request.TemplateID, request.StageIndex)
		if errors.Is(err, repository.ErrNotFound) {
			continue
		}
		if err != nil {
			return domain.ApprovalRequestPage{}, fmt.Errorf("resolve active stage: %w", err)
		}
		isHolder, err := s.store.IsActiveHolderOfPosition(ctx, identity.UserID, stage.PositionID)
		if err != nil {
			return domain.ApprovalRequestPage{}, fmt.Errorf("check active stage holder: %w", err)
		}
		if isHolder {
			matched = append(matched, request)
		}
	}

	start := (page - 1) * limit
	end := start + limit
	if start > len(matched) {
		start = len(matched)
	}
	if end > len(matched) {
		end = len(matched)
	}
	return domain.ApprovalRequestPage{Items: matched[start:end], Total: len(matched), Page: page, Limit: limit}, nil
}

// ListMonitoring backs §9.5 — HR/Top Management, org-wide, read-only.
func (s *DocumentApprovalService) ListMonitoring(
	ctx context.Context,
	identity domain.Identity,
	status *domain.DocumentApprovalStatus,
	templateID *uuid.UUID,
	page, limit int,
) (domain.ApprovalRequestPage, error) {
	if identity.Role != domain.RoleHR && identity.Role != domain.RoleTopManagement {
		return domain.ApprovalRequestPage{}, domain.ErrForbidden
	}
	page, limit = normalizePaging(page, limit)
	result, err := s.store.ListRequests(ctx, domain.ApprovalRequestFilter{
		Status: status, TemplateID: templateID, Page: page, Limit: limit,
	})
	if err != nil {
		return domain.ApprovalRequestPage{}, fmt.Errorf("list approval monitoring: %w", err)
	}
	return result, nil
}

// Detail backs §9.4. Read access: the requester, any past/current approver (to view
// history), or HR/Top Management. Non-access and non-existence both surface as
// ErrNotFound so existence doesn't leak, matching leave's Detail convention.
func (s *DocumentApprovalService) Detail(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
) (domain.ApprovalRequestDetail, error) {
	request, err := s.store.FindRequest(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.ApprovalRequestDetail{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ApprovalRequestDetail{}, fmt.Errorf("find approval request: %w", err)
	}

	template, stages, err := s.store.FindTemplate(ctx, request.TemplateID)
	if err != nil {
		return domain.ApprovalRequestDetail{}, fmt.Errorf("resolve template for detail: %w", err)
	}

	history, err := s.store.DecisionHistory(ctx, id)
	if err != nil {
		return domain.ApprovalRequestDetail{}, fmt.Errorf("load decision history: %w", err)
	}

	allowed, err := s.canRead(ctx, identity, request, history)
	if err != nil {
		return domain.ApprovalRequestDetail{}, err
	}
	if !allowed {
		return domain.ApprovalRequestDetail{}, domain.ErrNotFound
	}

	stageProgress := make([]domain.StageProgress, 0, len(stages))
	for _, stage := range stages {
		holders, err := s.store.ResolveStageApprovers(ctx, stage.PositionID)
		if err != nil {
			return domain.ApprovalRequestDetail{}, fmt.Errorf("resolve stage holders: %w", err)
		}
		stageProgress = append(stageProgress, domain.StageProgress{
			Order: stage.Order, StageName: stage.StageName, PositionID: stage.PositionID,
			Holders: holders,
			Done:    stage.Order < request.StageIndex || request.Status == domain.DocApprovalApproved,
			Active:  stage.Order == request.StageIndex && request.Status.Pending(),
		})
	}

	return domain.ApprovalRequestDetail{
		ApprovalRequest: request, TemplateName: template.Name, Stages: stageProgress, History: history,
	}, nil
}

func (s *DocumentApprovalService) canRead(
	ctx context.Context,
	identity domain.Identity,
	request domain.ApprovalRequest,
	history []domain.ApprovalDecisionRecord,
) (bool, error) {
	if request.RequesterUserID == identity.UserID {
		return true, nil
	}
	if identity.Role == domain.RoleHR || identity.Role == domain.RoleTopManagement {
		return true, nil
	}
	for _, decision := range history {
		if decision.ApproverUserID == identity.UserID {
			return true, nil
		}
	}
	// Not yet in history but might be the *current* active stage's approver.
	stage, err := s.store.StageAtOrder(ctx, request.TemplateID, request.StageIndex)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("resolve stage for read check: %w", err)
	}
	return s.store.IsActiveHolderOfPosition(ctx, identity.UserID, stage.PositionID)
}

// --- Decision --------------------------------------------------------------------------

// Decide implements §6.2/§5.3. Structurally mirrors leave's Decide (lock -> authorize ->
// conditional update -> append history -> audit -> event, all in one transaction), but the
// "what does approve actually do" branch is genuinely different: leave rewrites a status
// enum that encodes the stage; this advances an integer stage_index and only rewrites
// status on the terminal transition.
func (s *DocumentApprovalService) Decide(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	input domain.DecisionInput,
	meta RequestMeta,
) (domain.ApprovalRequest, error) {
	if !input.Approve && (input.Note == nil || strings.TrimSpace(*input.Note) == "") {
		return domain.ApprovalRequest{}, domain.ErrInvalidRequest
	}

	var result domain.ApprovalRequest
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		lock, err := s.store.LockRequestForDecision(txContext, id)
		if errors.Is(err, repository.ErrNotFound) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		if !lock.Status.Pending() {
			return domain.ErrAlreadyDecided
		}

		stage, err := s.store.StageAtOrder(txContext, lock.TemplateID, lock.StageIndex)
		if err != nil {
			return fmt.Errorf("resolve active stage: %w", err)
		}
		isHolder, err := s.store.IsActiveHolderOfPosition(txContext, identity.UserID, stage.PositionID)
		if err != nil {
			return fmt.Errorf("check decision authorization: %w", err)
		}
		if !isHolder {
			return domain.ErrForbidden
		}

		decision := domain.DocDecisionReject
		nextStatus := domain.DocApprovalRejected
		nextStageIndex := lock.StageIndex
		isFinal := lock.StageIndex == lock.TotalStages-1

		if input.Approve {
			decision = domain.DocDecisionApprove
			if isFinal {
				nextStatus = domain.DocApprovalApproved
			} else {
				nextStatus = domain.DocApprovalRunning
				nextStageIndex = lock.StageIndex + 1
			}
		}

		if input.Approve && !isFinal {
			if err := s.store.AdvanceStage(txContext, id, lock.StageIndex, nextStageIndex); err != nil {
				if errors.Is(err, repository.ErrConflict) {
					return domain.ErrAlreadyDecided
				}
				return err
			}
		} else {
			if err := s.store.FinalizeStatus(txContext, id, lock.StageIndex, lock.Status, nextStatus); err != nil {
				if errors.Is(err, repository.ErrConflict) {
					return domain.ErrAlreadyDecided
				}
				return err
			}
		}

		if err := s.store.AppendDecision(
			txContext, id, lock.StageIndex, identity.UserID, decision, input.Note,
		); err != nil {
			return err
		}

		var nextApprovers []domain.StageApprover
		if nextStatus == domain.DocApprovalRunning {
			nextStage, err := s.store.StageAtOrder(txContext, lock.TemplateID, nextStageIndex)
			if err != nil {
				return fmt.Errorf("resolve next stage: %w", err)
			}
			nextApprovers, err = s.store.ResolveStageApprovers(txContext, nextStage.PositionID)
			if err != nil {
				return fmt.Errorf("resolve next stage approvers: %w", err)
			}
			if len(nextApprovers) == 0 {
				// Same blocked-state handling as initiation (§13) — the newly-active
				// stage has no active holder.
				if err := s.store.FinalizeStatus(
					txContext, id, nextStageIndex, domain.DocApprovalRunning, domain.DocApprovalBlocked,
				); err != nil {
					return err
				}
				nextStatus = domain.DocApprovalBlocked
			}
		}

		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: map[bool]string{true: "APPROVE", false: "REJECT"}[input.Approve],
			Module: "persetujuan_dokumen",
			DataID: &id,
			Detail: map[string]any{
				"tahap_ke":    lock.StageIndex,
				"status_baru": string(nextStatus),
				"request_id":  meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		}); err != nil {
			return err
		}

		result = domain.ApprovalRequest{
			ID: id, TemplateID: lock.TemplateID, RequesterUserID: lock.RequesterUserID,
			StageIndex: nextStageIndex, Status: nextStatus,
		}
		// NOTE: notification dispatch deferred, same as Create — see follow-up task.
		return nil
	})
	if err != nil {
		return domain.ApprovalRequest{}, fmt.Errorf("decide approval request: %w", err)
	}
	return result, nil
}

// Cancel implements §9.4's cancel button rule and the PUT .../batalkan endpoint: only the
// requester, and only before any stage has decided (PRD wording) — enforced here as "zero
// decision rows exist yet," which is stricter than just "still at stage_index==0" since a
// blocked (bermasalah) request can also sit at stage 0 with zero decisions and should
// still be cancellable.
func (s *DocumentApprovalService) Cancel(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	meta RequestMeta,
) (domain.ApprovalRequest, error) {
	var result domain.ApprovalRequest
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		lock, err := s.store.LockRequestForDecision(txContext, id)
		if errors.Is(err, repository.ErrNotFound) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		if lock.RequesterUserID != identity.UserID {
			return domain.ErrForbidden
		}
		if !lock.Status.Pending() {
			return domain.ErrAlreadyDecided
		}
		decided, err := s.store.DecisionCount(txContext, id)
		if err != nil {
			return fmt.Errorf("count existing decisions: %w", err)
		}
		if decided > 0 {
			return domain.ErrAlreadyDecided
		}

		if err := s.store.FinalizeStatus(
			txContext, id, lock.StageIndex, lock.Status, domain.DocApprovalCancelled,
		); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return domain.ErrAlreadyDecided
			}
			return err
		}

		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "CANCEL",
			Module:    "persetujuan_dokumen",
			DataID:    &id,
			Detail:    map[string]any{"request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		}); err != nil {
			return err
		}

		result = domain.ApprovalRequest{
			ID: id, TemplateID: lock.TemplateID, RequesterUserID: lock.RequesterUserID,
			StageIndex: lock.StageIndex, Status: domain.DocApprovalCancelled,
		}
		// NOTE: notification dispatch deferred, same as Create/Decide.
		return nil
	})
	if err != nil {
		return domain.ApprovalRequest{}, fmt.Errorf("cancel approval request: %w", err)
	}
	return result, nil
}