package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

// OvertimeStore adalah kebutuhan penyimpanan lembur dari sisi service.
type OvertimeStore interface {
	CreateRequest(context.Context, domain.OvertimeRequestRow) (uuid.UUID, error)
	ListRequests(context.Context, domain.OvertimeRequestFilter) (domain.OvertimeRequestPage, error)
	FindRequest(context.Context, uuid.UUID) (domain.OvertimeRequestDetail, error)
	LockRequestForDecision(context.Context, uuid.UUID) (domain.RequestLock, error)
	UpdateRequestStatus(context.Context, uuid.UUID, domain.RequestStatus, domain.RequestStatus) error
	AppendApproval(
		context.Context, uuid.UUID, domain.ApprovalStage, *uuid.UUID,
		domain.ApprovalDecision, *string,
	) error
	Recap(context.Context, domain.OvertimeRecapFilter) ([]domain.OvertimeRecapItem, error)
}

type OvertimeService struct {
	overtimes   OvertimeStore
	supervisors SupervisorLookup
	tx          EmployeeTransactionManager
	audit       AuditWriter
	documents   DocumentStore
	events      domain.ApprovalEventPublisher
	now         func() time.Time
}

func NewOvertimeService(
	overtimes OvertimeStore,
	supervisors SupervisorLookup,
	tx EmployeeTransactionManager,
	audit AuditWriter,
	documents DocumentStore,
	events domain.ApprovalEventPublisher,
) *OvertimeService {
	return &OvertimeService{
		overtimes:   overtimes,
		supervisors: supervisors,
		tx:          tx,
		audit:       audit,
		documents:   documents,
		events:      events,
		now:         time.Now,
	}
}

// Create mengajukan lembur. Dokumen pendukung opsional, berbeda dengan ketidakhadiran.
// Durasi dihitung database dari jam mulai dan selesai.
func (s *OvertimeService) Create(
	ctx context.Context,
	identity domain.Identity,
	command domain.CreateOvertimeRequest,
	meta RequestMeta,
) (domain.RequestStateData, error) {
	supervisor, err := s.supervisors.SupervisorEmployeeID(ctx, identity.EmployeeID)
	if err != nil {
		return domain.RequestStateData{}, fmt.Errorf("resolve requester supervisor: %w", err)
	}
	status, ok := domain.InitialStatusForRole(identity.Role, supervisor != nil)
	if !ok {
		return domain.RequestStateData{}, domain.ErrForbidden
	}

	if _, err := time.ParseInLocation(domain.DateLayout, command.Date, domain.Jakarta()); err != nil {
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}
	start, err := parseOvertimeTime(command.StartTime)
	if err != nil {
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}
	end, err := parseOvertimeTime(command.EndTime)
	if err != nil || !end.After(start) {
		// Jam selesai harus setelah jam mulai; database juga menegakkannya lewat CHECK.
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}

	row := domain.OvertimeRequestRow{
		UserID:    identity.UserID,
		Date:      command.Date,
		StartTime: start.Format(domain.TimeLayout),
		EndTime:   end.Format(domain.TimeLayout),
		Reason:    strings.TrimSpace(command.Reason),
		Status:    status,
	}

	var objectPath string
	if command.Document != nil {
		objectPath = path.Join(
			"overtime-documents",
			identity.UserID.String(),
			uuid.NewString()+command.Document.Extension,
		)
		location, err := s.documents.Upload(
			ctx, objectPath,
			bytes.NewReader(command.Document.Content),
			command.Document.MediaType,
		)
		if err != nil {
			return domain.RequestStateData{}, fmt.Errorf("upload overtime document: %w", err)
		}
		row.DocumentURL = &location
	}

	var requestID uuid.UUID
	err = s.tx.Within(ctx, func(txContext context.Context) error {
		created, err := s.overtimes.CreateRequest(txContext, row)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		requestID = created
		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "CREATE",
			Module: "lembur",
			DataID: &created,
			Detail: map[string]any{
				"tanggal":    command.Date,
				"status":     string(status),
				"request_id": meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		}); err != nil {
			return err
		}
		initialStage, _ := domain.StageForStatus(status)
		return s.events.Publish(txContext, domain.ApprovalEvent{
			Type:            domain.EventOvertimeSubmitted,
			RequestID:       created,
			RequesterUserID: identity.UserID,
			ActorUserID:     &identity.UserID,
			Status:          status,
			Stage:           initialStage,
			NextStage:       domain.NextStageForStatus(status),
			OccurredAt:      s.now().UTC(),
		})
	})
	if err != nil {
		if objectPath != "" {
			if cleanupErr := s.documents.Delete(ctx, objectPath); cleanupErr != nil {
				slog.ErrorContext(ctx, "orphan overtime document cleanup failed",
					"object_path", objectPath, "error", cleanupErr)
			}
		}
		return domain.RequestStateData{}, fmt.Errorf("create overtime request: %w", err)
	}
	return domain.RequestStateData{ID: requestID, Status: status}, nil
}

// parseOvertimeTime menerima nilai input time dari browser (HH:MM) dan bentuk
// kontrak lengkap (HH:MM:SS), lalu service menyimpannya dalam bentuk kanonis.
func parseOvertimeTime(value string) (time.Time, error) {
	parsed, err := time.Parse(domain.TimeLayout, value)
	if err == nil {
		return parsed, nil
	}
	return time.Parse("15:04", value)
}

func (s *OvertimeService) List(
	ctx context.Context,
	identity domain.Identity,
	filter domain.OvertimeRequestFilter,
) (domain.OvertimeRequestPage, error) {
	scope, err := approvalScope(identity)
	if err != nil {
		return domain.OvertimeRequestPage{}, err
	}
	filter.Scope = scope
	filter.Page, filter.Limit = normalizePaging(filter.Page, filter.Limit)
	page, err := s.overtimes.ListRequests(ctx, filter)
	if err != nil {
		return domain.OvertimeRequestPage{}, fmt.Errorf("list overtime requests: %w", err)
	}
	return page, nil
}

// Detail mengembalikan satu pengajuan lembur bila identitas berhak membacanya.
func (s *OvertimeService) Detail(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
) (domain.OvertimeRequestDetail, error) {
	detail, err := s.overtimes.FindRequest(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.OvertimeRequestDetail{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.OvertimeRequestDetail{}, fmt.Errorf("find overtime request: %w", err)
	}

	if detail.EmployeeID == identity.EmployeeID {
		return detail, nil
	}
	switch identity.Role {
	case domain.RoleHR, domain.RoleTopManagement:
		return detail, nil
	case domain.RoleSupervisor:
		supervisor, err := s.supervisors.SupervisorEmployeeID(ctx, detail.EmployeeID)
		if err != nil {
			return domain.OvertimeRequestDetail{}, fmt.Errorf("resolve request supervisor: %w", err)
		}
		if supervisor != nil && *supervisor == identity.EmployeeID {
			return detail, nil
		}
	}
	// Ketidakberhakan dan ketidakadaan menghasilkan error yang sama.
	return domain.OvertimeRequestDetail{}, domain.ErrNotFound
}

// Decide menerapkan keputusan lembur dengan aturan konkurensi yang sama seperti
// ketidakhadiran. Tidak ada saldo yang dikurangi dan tidak ada kompensasi yang dihitung.
func (s *OvertimeService) Decide(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	input domain.DecisionInput,
	meta RequestMeta,
) (domain.RequestStateData, error) {
	if !input.Approve && (input.Note == nil || strings.TrimSpace(*input.Note) == "") {
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}

	var result domain.RequestStateData
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		lock, err := s.overtimes.LockRequestForDecision(txContext, id)
		if errors.Is(err, repository.ErrNotFound) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		if !lock.Status.Pending() {
			return domain.ErrAlreadyDecided
		}
		if !domain.CanDecide(identity.Role, lock.Status) {
			return domain.ErrForbidden
		}
		if identity.Role == domain.RoleSupervisor &&
			(lock.SupervisorEmployeeID == nil || *lock.SupervisorEmployeeID != identity.EmployeeID) {
			return domain.ErrForbidden
		}
		stage, _ := domain.StageForStatus(lock.Status)

		next := domain.StatusRejected
		decision := domain.DecisionReject
		if input.Approve {
			decision = domain.DecisionApprove
			resolved, ok := domain.NextStatusAfterApprove(lock.Status)
			if !ok {
				return domain.ErrAlreadyDecided
			}
			next = resolved
		}

		if err := s.overtimes.UpdateRequestStatus(txContext, id, lock.Status, next); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return domain.ErrAlreadyDecided
			}
			return err
		}
		if err := s.overtimes.AppendApproval(
			txContext, id, stage, &identity.UserID, decision, input.Note,
		); err != nil {
			return err
		}
		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: map[bool]string{true: "APPROVE", false: "REJECT"}[input.Approve],
			Module: "lembur",
			DataID: &id,
			Detail: map[string]any{
				"tahap":       string(stage),
				"status_baru": string(next),
				"request_id":  meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		}); err != nil {
			return err
		}
		result = domain.RequestStateData{ID: id, Status: next}
		return s.events.Publish(txContext, domain.ApprovalEvent{
			Type:            domain.EventOvertimeDecisionChanged,
			RequestID:       id,
			RequesterUserID: lock.RequesterUserID,
			ActorUserID:     &identity.UserID,
			Status:          next,
			Stage:           stage,
			NextStage:       domain.NextStageForStatus(next),
			OccurredAt:      s.now().UTC(),
		})
	})
	if err != nil {
		return domain.RequestStateData{}, fmt.Errorf("decide overtime request: %w", err)
	}
	return result, nil
}

// Recap hanya untuk HR sesuai API Contract.
func (s *OvertimeService) Recap(
	ctx context.Context,
	identity domain.Identity,
	filter domain.OvertimeRecapFilter,
) ([]domain.OvertimeRecapItem, error) {
	if identity.Role != domain.RoleHR {
		return nil, domain.ErrForbidden
	}
	items, err := s.overtimes.Recap(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("read overtime recap: %w", err)
	}
	return items, nil
}
