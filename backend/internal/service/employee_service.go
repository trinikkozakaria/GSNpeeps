package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

type EmployeeReader interface {
	ListDepartments(context.Context) ([]domain.Department, error)
	ListPositions(context.Context, *uuid.UUID) ([]domain.Position, error)
	List(context.Context, domain.EmployeeFilter) (domain.EmployeePage, error)
	FindByID(context.Context, uuid.UUID) (domain.EmployeeDetail, error)
	ValidateCreate(context.Context, domain.CreateEmployee) error
	Create(context.Context, domain.CreateEmployee) (domain.EmployeeMutationResult, error)
	ValidateMutation(context.Context, uuid.UUID, domain.EmployeeChanges) error
	Update(context.Context, uuid.UUID, domain.EmployeeChanges) (domain.EmployeeMutationResult, error)
	SoftDelete(context.Context, uuid.UUID) (domain.EmployeeMutationResult, error)
}

type EmployeeTransactionManager interface {
	Within(context.Context, func(context.Context) error) error
}

type SessionRevoker interface {
	Revoke(context.Context, uuid.UUID) error
}

type EmployeeService struct {
	employees EmployeeReader
	tx        EmployeeTransactionManager
	audit     AuditWriter
	sessions  SessionRevoker
	passwords PasswordHasher
	now       func() time.Time
}

func NewEmployeeService(
	employees EmployeeReader,
	tx EmployeeTransactionManager,
	audit AuditWriter,
	sessions SessionRevoker,
	passwords PasswordHasher,
) *EmployeeService {
	return &EmployeeService{
		employees: employees,
		tx:        tx,
		audit:     audit,
		sessions:  sessions,
		passwords: passwords,
		now:       time.Now,
	}
}

func (s *EmployeeService) Create(
	ctx context.Context,
	identity domain.Identity,
	request dto.CreateEmployeeRequest,
	meta RequestMeta,
) (domain.EmployeeMutationResult, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeeMutationResult{}, domain.ErrForbidden
	}
	startDate, err := time.Parse("2006-01-02", request.Contract.StartDate)
	if err != nil {
		return domain.EmployeeMutationResult{}, domain.ErrInvalidRequest
	}
	endDate, err := time.Parse("2006-01-02", request.Contract.EndDate)
	if err != nil || endDate.Before(startDate) {
		return domain.EmployeeMutationResult{}, domain.ErrInvalidRequest
	}
	rawCredential, err := randomLockedCredential()
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("generate locked account credential: %w", err)
	}
	passwordHash, err := s.passwords.Hash(rawCredential)
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("hash locked account credential: %w", err)
	}
	command := createEmployeeCommand(request, passwordHash)
	var result domain.EmployeeMutationResult
	err = s.tx.Within(ctx, func(txContext context.Context) error {
		if err := s.employees.ValidateCreate(txContext, command); err != nil {
			return mapEmployeeRepositoryError(err)
		}
		created, err := s.employees.Create(txContext, command)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		result = created
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID: &identity.UserID,
			Action: "CREATE",
			Module: "karyawan",
			DataID: &created.EmployeeID,
			Detail: map[string]any{
				"department_id": command.DepartmentID,
				"position_id":   command.PositionID,
				"role":          command.Role,
				"account_state": "locked_pending_activation",
				"request_id":    meta.RequestID,
			},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("create employee: %w", err)
	}
	return result, nil
}

func createEmployeeCommand(
	request dto.CreateEmployeeRequest,
	passwordHash string,
) domain.CreateEmployee {
	return domain.CreateEmployee{
		NIP:           strings.TrimSpace(request.NIP),
		Name:          strings.TrimSpace(request.Name),
		Email:         strings.ToLower(strings.TrimSpace(request.Email)),
		PasswordHash:  passwordHash,
		Gender:        request.Gender,
		BirthDate:     request.BirthDate,
		JoinDate:      request.JoinDate,
		DepartmentID:  request.DepartmentID,
		PositionID:    request.PositionID,
		SupervisorID:  request.SupervisorID,
		MaritalStatus: request.MaritalStatus,
		Role:          request.Role,
		Address: domain.EmployeeAddress{
			Street:   strings.TrimSpace(request.Address.Street),
			Village:  trimOptionalString(request.Address.Village),
			District: trimOptionalString(request.Address.District),
			City:     strings.TrimSpace(request.Address.City),
			Province: strings.TrimSpace(request.Address.Province),
		},
		KTPNumber: strings.TrimSpace(request.KTP.Number),
		Contract: domain.CreateEmployeeContract{
			Number:    strings.TrimSpace(request.Contract.Number),
			Type:      request.Contract.Type,
			StartDate: request.Contract.StartDate,
			EndDate:   request.Contract.EndDate,
		},
	}
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func randomLockedCredential() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (s *EmployeeService) Update(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	request dto.UpdateEmployeeRequest,
	meta RequestMeta,
) (domain.EmployeeMutationResult, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeeMutationResult{}, domain.ErrForbidden
	}
	if request.Empty() {
		return domain.EmployeeMutationResult{}, domain.ErrInvalidRequest
	}
	changes := employeeChanges(request)
	var result domain.EmployeeMutationResult
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		if err := s.employees.ValidateMutation(txContext, id, changes); err != nil {
			return mapEmployeeRepositoryError(err)
		}
		updated, err := s.employees.Update(txContext, id, changes)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		result = updated
		if requiresSessionRevocation(changes) && updated.UserID != nil {
			if err := s.sessions.Revoke(txContext, *updated.UserID); err != nil {
				return fmt.Errorf("revoke employee session: %w", err)
			}
		}
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "UPDATE",
			Module:    "karyawan",
			DataID:    &id,
			Detail:    map[string]any{"fields": changedEmployeeFields(changes), "request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("update employee: %w", err)
	}
	return result, nil
}

func (s *EmployeeService) Deactivate(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
	meta RequestMeta,
) (domain.EmployeeMutationResult, error) {
	if identity.Role != domain.RoleHR {
		return domain.EmployeeMutationResult{}, domain.ErrForbidden
	}
	var result domain.EmployeeMutationResult
	err := s.tx.Within(ctx, func(txContext context.Context) error {
		deactivated, err := s.employees.SoftDelete(txContext, id)
		if err != nil {
			return mapEmployeeRepositoryError(err)
		}
		result = deactivated
		if deactivated.UserID != nil {
			if err := s.sessions.Revoke(txContext, *deactivated.UserID); err != nil {
				return fmt.Errorf("revoke deactivated employee session: %w", err)
			}
		}
		return s.audit.Append(txContext, domain.AuditEntry{
			UserID:    &identity.UserID,
			Action:    "DELETE",
			Module:    "karyawan",
			DataID:    &id,
			Detail:    map[string]any{"status": "nonaktif", "request_id": meta.RequestID},
			IPAddress: meta.IPAddress,
			CreatedAt: s.now().UTC(),
		})
	})
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("deactivate employee: %w", err)
	}
	return result, nil
}

func employeeChanges(request dto.UpdateEmployeeRequest) domain.EmployeeChanges {
	changes := domain.EmployeeChanges{
		Name:          request.Name,
		Email:         request.Email,
		Gender:        request.Gender,
		BirthDate:     request.BirthDate,
		JoinDate:      request.JoinDate,
		DepartmentID:  request.DepartmentID,
		PositionID:    request.PositionID,
		SupervisorID:  request.SupervisorID.Value,
		SupervisorSet: request.SupervisorID.Set,
		MaritalStatus: request.MaritalStatus,
		Status:        request.Status,
		Role:          request.Role,
	}
	if changes.Name != nil {
		normalized := strings.TrimSpace(*changes.Name)
		changes.Name = &normalized
	}
	if changes.Email != nil {
		normalized := strings.ToLower(strings.TrimSpace(*changes.Email))
		changes.Email = &normalized
	}
	return changes
}

func mapEmployeeRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return domain.ErrNotFound
	case errors.Is(err, repository.ErrConflict):
		return domain.ErrConflict
	default:
		return err
	}
}

func requiresSessionRevocation(changes domain.EmployeeChanges) bool {
	return changes.Email != nil || changes.Role != nil || changes.Status != nil
}

func changedEmployeeFields(changes domain.EmployeeChanges) []string {
	fields := make([]string, 0, 11)
	candidates := []struct {
		name    string
		changed bool
	}{
		{"nama", changes.Name != nil},
		{"email", changes.Email != nil},
		{"jenis_kelamin", changes.Gender != nil},
		{"tanggal_lahir", changes.BirthDate != nil},
		{"tanggal_join", changes.JoinDate != nil},
		{"department_id", changes.DepartmentID != nil},
		{"position_id", changes.PositionID != nil},
		{"atasan_id", changes.SupervisorSet},
		{"status_pernikahan", changes.MaritalStatus != nil},
		{"status", changes.Status != nil},
		{"role", changes.Role != nil},
	}
	for _, candidate := range candidates {
		if candidate.changed {
			fields = append(fields, candidate.name)
		}
	}
	return fields
}

func (s *EmployeeService) ListDepartments(ctx context.Context) ([]domain.Department, error) {
	items, err := s.employees.ListDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list department master: %w", err)
	}
	return items, nil
}

func (s *EmployeeService) ListPositions(
	ctx context.Context,
	departmentID *uuid.UUID,
) ([]domain.Position, error) {
	items, err := s.employees.ListPositions(ctx, departmentID)
	if err != nil {
		return nil, fmt.Errorf("list position master: %w", err)
	}
	return items, nil
}

func (s *EmployeeService) List(
	ctx context.Context,
	identity domain.Identity,
	filter domain.EmployeeFilter,
) (domain.EmployeePage, error) {
	if identity.Role != domain.RoleHR && identity.Role != domain.RoleTopManagement {
		return domain.EmployeePage{}, domain.ErrForbidden
	}
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Status != "" && filter.Status != "aktif" && filter.Status != "nonaktif" {
		return domain.EmployeePage{}, domain.ErrInvalidRequest
	}
	page, err := s.employees.List(ctx, filter)
	if err != nil {
		return domain.EmployeePage{}, fmt.Errorf("list employee database: %w", err)
	}
	return page, nil
}

func (s *EmployeeService) Detail(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
) (domain.EmployeeDetail, error) {
	if identity.Role != domain.RoleHR && identity.Role != domain.RoleTopManagement {
		return domain.EmployeeDetail{}, domain.ErrForbidden
	}
	item, err := s.employees.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.EmployeeDetail{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.EmployeeDetail{}, fmt.Errorf("get employee detail: %w", err)
	}
	return item, nil
}
