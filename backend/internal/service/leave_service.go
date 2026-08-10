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

// LeaveStore adalah kebutuhan penyimpanan ketidakhadiran dari sisi service.
type LeaveStore interface {
	ListLeaveTypes(context.Context, *bool) ([]domain.LeaveType, error)
	FindLeaveType(context.Context, uuid.UUID) (domain.LeaveType, error)
	CreateLeaveType(context.Context, domain.CreateLeaveType) (uuid.UUID, error)
	UpdateLeaveType(context.Context, uuid.UUID, domain.UpdateLeaveType) error

	CreateRequest(context.Context, domain.LeaveRequestRow) (uuid.UUID, error)
	ListRequests(context.Context, domain.LeaveRequestFilter) (domain.LeaveRequestPage, error)
	FindRequest(context.Context, uuid.UUID) (domain.LeaveRequestDetail, error)

	LockRequestForDecision(context.Context, uuid.UUID) (domain.RequestLock, error)
	UpdateRequestStatus(context.Context, uuid.UUID, domain.RequestStatus, domain.RequestStatus) error
	AppendApproval(
		context.Context, uuid.UUID, domain.ApprovalStage, *uuid.UUID,
		domain.ApprovalDecision, *string,
	) error
	DeductLeaveBalance(context.Context, uuid.UUID, int, int) error
}

// SupervisorLookup menyediakan relasi atasan langsung untuk routing dan otorisasi.
type SupervisorLookup interface {
	SupervisorEmployeeID(context.Context, uuid.UUID) (*uuid.UUID, error)
}

type LeaveService struct {
	leaves      LeaveStore
	supervisors SupervisorLookup
	tx          EmployeeTransactionManager
	audit       AuditWriter
	documents   DocumentStore
	events      domain.ApprovalEventPublisher
	now         func() time.Time
}

func NewLeaveService(
	leaves LeaveStore,
	supervisors SupervisorLookup,
	tx EmployeeTransactionManager,
	audit AuditWriter,
	documents DocumentStore,
	events domain.ApprovalEventPublisher,
) *LeaveService {
	return &LeaveService{
		leaves:      leaves,
		supervisors: supervisors,
		tx:          tx,
		audit:       audit,
		documents:   documents,
		events:      events,
		now:         time.Now,
	}
}

// ListLeaveTypes terbuka untuk seluruh role terautentikasi karena pemohon membutuhkan
// jenis_izin_id saat mengajukan ketidakhadiran (resolusi G-012). Hanya HR yang boleh
// melihat master nonaktif; role lain selalu dipaksa ke daftar aktif agar jenis izin yang
// sengaja dinonaktifkan tidak dapat diajukan. Administrasi master tetap HR-only.
func (s *LeaveService) ListLeaveTypes(
	ctx context.Context,
	identity domain.Identity,
	activeOnly *bool,
) ([]domain.LeaveType, error) {
	if identity.Role != domain.RoleHR {
		forced := true
		activeOnly = &forced
	}
	items, err := s.leaves.ListLeaveTypes(ctx, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("list leave types: %w", err)
	}
	return items, nil
}

func (s *LeaveService) CreateLeaveType(
	ctx context.Context,
	identity domain.Identity,
	command domain.CreateLeaveType,
	meta RequestMeta,
) (uuid.UUID, error) {
	if identity.Role != domain.RoleHR {
		return uuid.Nil, domain.ErrForbidden
	}
	command.Code = strings.TrimSpace(command.Code)
	command.Name = strings.TrimSpace(command.Name)
	if command.Code == "" || command.Name == "" || command.AnnualQuota < 0 {
		return uuid.Nil, domain.ErrInvalidRequest
	}

	var id uuid.UUID
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		created, err := s.leaves.CreateLeaveType(txContext, command)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		id = created
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "CREATE",
			Module: "master_jenis_izin",
			DataID: &created,
			Detail: map[string]any{
				"kode":          command.Code,
				"kuota_tahunan": command.AnnualQuota,
				"request_id":    meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create leave type: %w", err)
	}
	return id, nil
}

func (s *LeaveService) UpdateLeaveType(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	changes domain.UpdateLeaveType,
	meta RequestMeta,
) error {
	if identity.Role != domain.RoleHR {
		return domain.ErrForbidden
	}
	if changes.Empty() {
		return domain.ErrInvalidRequest
	}
	if changes.AnnualQuota != nil && *changes.AnnualQuota < 0 {
		return domain.ErrInvalidRequest
	}

	err := s.tx.Within(ctx, func(txContext context.Context) error {
		if err := s.leaves.UpdateLeaveType(txContext, id, changes); err != nil {
			return mapEmployeeRepositoryError(err)
		}
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "UPDATE",
			Module:    "master_jenis_izin",
			DataID:    &id,
			Detail:    map[string]any{"request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return fmt.Errorf("update leave type: %w", err)
	}
	return nil
}

// Create mengajukan ketidakhadiran. Saldo tidak dikurangi saat submit; pengurangan hanya
// terjadi pada final approval.
func (s *LeaveService) Create(
	ctx context.Context,
	identity domain.Identity,
	command domain.CreateLeaveRequest,
	meta RequestMeta,
) (domain.RequestStateData, error) {
	supervisor, err := s.supervisors.SupervisorEmployeeID(ctx, identity.EmployeeID)
	if err != nil {
		return domain.RequestStateData{}, fmt.Errorf("resolve requester supervisor: %w", err)
	}
	status, ok := domain.InitialStatusForRole(identity.Role, supervisor != nil)
	if !ok {
		// Top Management tidak memiliki jalur approval pada kontrak.
		return domain.RequestStateData{}, domain.ErrForbidden
	}

	start, err := time.ParseInLocation(domain.DateLayout, command.StartDate, domain.Jakarta())
	if err != nil {
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}
	end, err := time.ParseInLocation(domain.DateLayout, command.EndDate, domain.Jakarta())
	if err != nil || end.Before(start) {
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}
	totalDays := domain.TotalLeaveDays(start, end)

	leaveType, err := s.leaves.FindLeaveType(ctx, command.LeaveTypeID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.RequestStateData{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RequestStateData{}, fmt.Errorf("resolve leave type: %w", err)
	}
	if !leaveType.IsActive {
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}
	// Kewajiban dokumen ditentukan master jenis izin (D-024).
	if leaveType.RequiresDocument && command.Document == nil {
		return domain.RequestStateData{}, domain.ErrDocumentRequired
	}
	// Kuota tahunan membatasi panjang satu pengajuan; saldo aktual diperiksa saat final
	// approval agar pengajuan tidak menahan saldo lebih awal.
	if leaveType.AnnualQuota > 0 && totalDays > leaveType.AnnualQuota {
		return domain.RequestStateData{}, domain.ErrInsufficientLeaveBalance
	}

	row := domain.LeaveRequestRow{
		UserID:          identity.UserID,
		LeaveTypeID:     leaveType.ID,
		StartDate:       command.StartDate,
		EndDate:         command.EndDate,
		TotalDays:       totalDays,
		Reason:          strings.TrimSpace(command.Reason),
		Destination:     command.Destination,
		DestinationNote: command.DestinationNote,
		Status:          status,
	}

	var objectPath string
	if command.Document != nil {
		objectPath = path.Join(
			"leave-documents",
			identity.UserID.String(),
			uuid.NewString()+command.Document.Extension,
		)
		location, err := s.documents.Upload(
			ctx, objectPath,
			bytes.NewReader(command.Document.Content),
			command.Document.MediaType,
		)
		if err != nil {
			return domain.RequestStateData{}, fmt.Errorf("upload leave document: %w", err)
		}
		row.DocumentURL = &location
	}

	var requestID uuid.UUID
	err = s.tx.Within(ctx, func(txContext context.Context) error {
		created, err := s.leaves.CreateRequest(txContext, row)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		requestID = created
		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "CREATE",
			Module: "ketidakhadiran",
			DataID: &created,
			Detail: map[string]any{
				"jenis_izin_id": leaveType.ID,
				"jumlah_hari":   totalDays,
				"status":        string(status),
				"request_id":    meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		}); err != nil {
			return err
		}
		// Event dipublikasikan di dalam transaction agar kegagalan publikasi membatalkan
		// pengajuan dan tidak menghasilkan side effect yang hilang diam-diam.
		initialStage, _ := domain.StageForStatus(status)
		return s.events.Publish(txContext, domain.ApprovalEvent{
			Type:            domain.EventLeaveSubmitted,
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
				slog.ErrorContext(ctx, "orphan leave document cleanup failed",
					"object_path", objectPath, "error", cleanupErr)
			}
		}
		return domain.RequestStateData{}, fmt.Errorf("create leave request: %w", err)
	}
	return domain.RequestStateData{ID: requestID, Status: status}, nil
}

// ListForApproval mengembalikan inbox approver sesuai tahap dan relasi yang diizinkan.
func (s *LeaveService) ListForApproval(
	ctx context.Context,
	identity domain.Identity,
	status *domain.RequestStatus,
	page, limit int,
) (domain.LeaveRequestPage, error) {
	scope, err := approvalScope(identity)
	if err != nil {
		return domain.LeaveRequestPage{}, err
	}
	return s.list(ctx, domain.LeaveRequestFilter{
		Scope: scope, Status: status, Page: page, Limit: limit,
	})
}

// ListMine mengembalikan pengajuan milik pemohon sendiri.
func (s *LeaveService) ListMine(
	ctx context.Context,
	identity domain.Identity,
	page, limit int,
) (domain.LeaveRequestPage, error) {
	if identity.Role == domain.RoleTopManagement {
		return domain.LeaveRequestPage{}, domain.ErrForbidden
	}
	userID := identity.UserID
	return s.list(ctx, domain.LeaveRequestFilter{
		Scope: domain.LeaveRequestScope{RequesterUserID: &userID},
		Page:  page,
		Limit: limit,
	})
}

func (s *LeaveService) list(
	ctx context.Context,
	filter domain.LeaveRequestFilter,
) (domain.LeaveRequestPage, error) {
	filter.Page, filter.Limit = normalizePaging(filter.Page, filter.Limit)
	page, err := s.leaves.ListRequests(ctx, filter)
	if err != nil {
		return domain.LeaveRequestPage{}, fmt.Errorf("list leave requests: %w", err)
	}
	return page, nil
}

// approvalScope membatasi inbox: Atasan hanya bawahan langsung pada tahap atasan, HR hanya
// tahap HR, dan Top Management hanya pengajuan milik HR pada tahap Top Management.
func approvalScope(identity domain.Identity) (domain.LeaveRequestScope, error) {
	switch identity.Role {
	case domain.RoleSupervisor:
		employeeID := identity.EmployeeID
		stage := domain.StatusWaitingSupervisor
		return domain.LeaveRequestScope{
			SupervisorEmployeeID: &employeeID,
			Stage:                &stage,
		}, nil
	case domain.RoleHR:
		stage := domain.StatusWaitingHR
		return domain.LeaveRequestScope{Stage: &stage}, nil
	case domain.RoleTopManagement:
		stage := domain.StatusWaitingTopManagement
		role := domain.RoleHR
		return domain.LeaveRequestScope{Stage: &stage, RequesterRole: &role}, nil
	default:
		return domain.LeaveRequestScope{}, domain.ErrForbidden
	}
}

func normalizePaging(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// Detail mengembalikan satu pengajuan bila identitas berhak membacanya. Ketidakberhakan dan
// ketidakadaan sama-sama menghasilkan ErrNotFound agar keberadaan record tidak bocor.
func (s *LeaveService) Detail(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
) (domain.LeaveRequestDetail, error) {
	detail, err := s.leaves.FindRequest(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.LeaveRequestDetail{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.LeaveRequestDetail{}, fmt.Errorf("find leave request: %w", err)
	}

	allowed, err := s.canRead(ctx, identity, detail)
	if err != nil {
		return domain.LeaveRequestDetail{}, err
	}
	if !allowed {
		return domain.LeaveRequestDetail{}, domain.ErrNotFound
	}
	return detail, nil
}

func (s *LeaveService) canRead(
	ctx context.Context,
	identity domain.Identity,
	detail domain.LeaveRequestDetail,
) (bool, error) {
	// Pemohon selalu boleh membaca pengajuan sendiri.
	if detail.EmployeeID == identity.EmployeeID {
		return true, nil
	}
	switch identity.Role {
	case domain.RoleHR:
		return true, nil
	case domain.RoleTopManagement:
		return true, nil
	case domain.RoleSupervisor:
		supervisor, err := s.supervisors.SupervisorEmployeeID(ctx, detail.EmployeeID)
		if err != nil {
			return false, fmt.Errorf("resolve request supervisor: %w", err)
		}
		return supervisor != nil && *supervisor == identity.EmployeeID, nil
	default:
		return false, nil
	}
}

// Decide menerapkan persetujuan atau penolakan pada tahap aktif.
//
// Seluruh langkah berjalan dalam satu transaction: baris pengajuan dikunci, tahap dan
// approver diverifikasi, status diubah secara kondisional, riwayat ditulis, saldo dikurangi
// bila final, lalu audit dan event dipublikasikan. Dua keputusan bersaing diserialkan oleh
// row lock sehingga hanya satu yang menang; yang kalah menerima ErrAlreadyDecided.
func (s *LeaveService) Decide(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	input domain.DecisionInput,
	meta RequestMeta,
) (domain.RequestStateData, error) {
	if !input.Approve && (input.Note == nil || strings.TrimSpace(*input.Note) == "") {
		// Penolakan wajib memiliki catatan.
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}

	var result domain.RequestStateData
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		lock, err := s.leaves.LockRequestForDecision(txContext, id)
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
		// Atasan hanya boleh memutus pengajuan bawahan langsungnya.
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

		if err := s.leaves.UpdateRequestStatus(txContext, id, lock.Status, next); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return domain.ErrAlreadyDecided
			}
			return err
		}
		if err := s.leaves.AppendApproval(
			txContext, id, stage, &identity.UserID, decision, input.Note,
		); err != nil {
			return err
		}
		// Saldo hanya berkurang sekali, tepat pada final approval. Penolakan tidak
		// mengurangi saldo.
		if next == domain.StatusApproved && lock.AnnualQuota > 0 {
			if err := s.leaves.DeductLeaveBalance(
				txContext, lock.RequesterUserID, lock.Year, lock.TotalDays,
			); err != nil {
				if errors.Is(err, repository.ErrConflict) {
					return domain.ErrInsufficientLeaveBalance
				}
				return err
			}
		}
		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: map[bool]string{true: "APPROVE", false: "REJECT"}[input.Approve],
			Module: "ketidakhadiran",
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
			Type:            domain.EventLeaveDecisionChanged,
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
		return domain.RequestStateData{}, fmt.Errorf("decide leave request: %w", err)
	}
	return result, nil
}

// Delegate memindahkan keputusan dari Atasan ke HR. Hanya Atasan yang sedang menjadi
// approver pengajuan bawahannya yang boleh mendelegasikan.
func (s *LeaveService) Delegate(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	note string,
	meta RequestMeta,
) (domain.RequestStateData, error) {
	if identity.Role != domain.RoleSupervisor {
		return domain.RequestStateData{}, domain.ErrForbidden
	}
	if strings.TrimSpace(note) == "" {
		return domain.RequestStateData{}, domain.ErrInvalidRequest
	}

	var result domain.RequestStateData
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		lock, err := s.leaves.LockRequestForDecision(txContext, id)
		if errors.Is(err, repository.ErrNotFound) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		// Delegasi hanya sah pada tahap Atasan; setelah eskalasi atau keputusan lain
		// status sudah berubah dan operasi ini kalah bersaing.
		if lock.Status != domain.StatusWaitingSupervisor {
			return domain.ErrAlreadyDecided
		}
		if lock.SupervisorEmployeeID == nil || *lock.SupervisorEmployeeID != identity.EmployeeID {
			return domain.ErrForbidden
		}

		if err := s.leaves.UpdateRequestStatus(
			txContext, id, domain.StatusWaitingSupervisor, domain.StatusWaitingHR,
		); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return domain.ErrAlreadyDecided
			}
			return err
		}
		if err := s.leaves.AppendApproval(
			txContext, id, domain.StageSupervisor, &identity.UserID,
			domain.DecisionDelegate, &note,
		); err != nil {
			return err
		}
		if err := s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "DELEGATE",
			Module:    "ketidakhadiran",
			DataID:    &id,
			Detail:    map[string]any{"status_baru": string(domain.StatusWaitingHR), "request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		}); err != nil {
			return err
		}
		result = domain.RequestStateData{ID: id, Status: domain.StatusWaitingHR}
		return s.events.Publish(txContext, domain.ApprovalEvent{
			Type:            domain.EventLeaveDelegated,
			RequestID:       id,
			RequesterUserID: lock.RequesterUserID,
			ActorUserID:     &identity.UserID,
			Status:          domain.StatusWaitingHR,
			Stage:           domain.StageSupervisor,
			NextStage:       domain.NextStageForStatus(domain.StatusWaitingHR),
			OccurredAt:      s.now().UTC(),
		})
	})
	if err != nil {
		return domain.RequestStateData{}, fmt.Errorf("delegate leave request: %w", err)
	}
	return result, nil
}
