package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type employeeReaderStub struct {
	receivedFilter domain.EmployeeFilter
	detailError    error
	createCommand  domain.CreateEmployee
}

func (s *employeeReaderStub) ValidateCreate(context.Context, domain.CreateEmployee) error {
	return nil
}

func (s *employeeReaderStub) Create(
	_ context.Context,
	command domain.CreateEmployee,
) (domain.EmployeeMutationResult, error) {
	s.createCommand = command
	return domain.EmployeeMutationResult{EmployeeID: uuid.New()}, nil
}

func (s *employeeReaderStub) ValidateMutation(
	context.Context,
	uuid.UUID,
	domain.EmployeeChanges,
) error {
	return nil
}

func (s *employeeReaderStub) Update(
	context.Context,
	uuid.UUID,
	domain.EmployeeChanges,
) (domain.EmployeeMutationResult, error) {
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeReaderStub) SoftDelete(
	context.Context,
	uuid.UUID,
) (domain.EmployeeMutationResult, error) {
	return domain.EmployeeMutationResult{}, nil
}

type transactionStub struct{}

func (transactionStub) Within(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type auditStub struct{}

func (auditStub) Append(context.Context, domain.AuditEntry) error { return nil }

type sessionRevokerStub struct{}

func (sessionRevokerStub) Revoke(context.Context, uuid.UUID) error { return nil }

type passwordHasherStub struct{}

func (passwordHasherStub) Hash(string) (string, error)         { return "locked-hash", nil }
func (passwordHasherStub) Verify(string, string) (bool, error) { return false, nil }

func newEmployeeServiceForTest(reader EmployeeReader) *EmployeeService {
	return NewEmployeeService(
		reader,
		transactionStub{},
		auditStub{},
		sessionRevokerStub{},
		passwordHasherStub{},
	)
}

func (s *employeeReaderStub) ListDepartments(context.Context) ([]domain.Department, error) {
	return []domain.Department{}, nil
}

func (s *employeeReaderStub) ListPositions(
	context.Context,
	*uuid.UUID,
) ([]domain.Position, error) {
	return []domain.Position{}, nil
}

func (s *employeeReaderStub) List(
	_ context.Context,
	filter domain.EmployeeFilter,
) (domain.EmployeePage, error) {
	s.receivedFilter = filter
	return domain.EmployeePage{Items: []domain.EmployeeSummary{}, Page: filter.Page, Limit: filter.Limit}, nil
}

func (s *employeeReaderStub) FindByID(
	context.Context,
	uuid.UUID,
) (domain.EmployeeDetail, error) {
	return domain.EmployeeDetail{}, s.detailError
}

func TestEmployeeListRestrictsRoleAndAppliesBounds(t *testing.T) {
	stub := &employeeReaderStub{}
	service := newEmployeeServiceForTest(stub)

	_, err := service.List(context.Background(), domain.Identity{Role: domain.RoleEmployee}, domain.EmployeeFilter{})
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = service.List(context.Background(), domain.Identity{Role: domain.RoleHR}, domain.EmployeeFilter{
		Page:  -1,
		Limit: 999,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, stub.receivedFilter.Page)
	assert.Equal(t, 100, stub.receivedFilter.Limit)
}

func TestEmployeeListRejectsUnknownStatus(t *testing.T) {
	service := newEmployeeServiceForTest(&employeeReaderStub{})

	_, err := service.List(context.Background(), domain.Identity{Role: domain.RoleHR}, domain.EmployeeFilter{
		Status: "resign",
	})

	require.ErrorIs(t, err, domain.ErrInvalidRequest)
}

func TestEmployeeDetailMapsRepositoryNotFound(t *testing.T) {
	service := newEmployeeServiceForTest(&employeeReaderStub{detailError: repository.ErrNotFound})

	_, err := service.Detail(
		context.Background(),
		domain.Identity{Role: domain.RoleTopManagement},
		uuid.New(),
	)

	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestEmployeeCreateRequiresHRAndBuildsLockedAccount(t *testing.T) {
	stub := &employeeReaderStub{}
	service := newEmployeeServiceForTest(stub)
	request := dto.CreateEmployeeRequest{
		NIP:          " EMP-001 ",
		Name:         " Karyawan Uji ",
		Email:        " EMPLOYEE@EXAMPLE.TEST ",
		Gender:       "P",
		BirthDate:    "1995-04-10",
		JoinDate:     "2026-07-29",
		DepartmentID: uuid.New(),
		PositionID:   uuid.New(),
		Role:         domain.RoleEmployee,
		Address: dto.EmployeeAddressRequest{
			Street:   "Jalan Sintetis",
			City:     "Jakarta",
			Province: "DKI Jakarta",
		},
		KTP: dto.EmployeeKTPRequest{Number: "3174000000000001"},
		Contract: dto.EmployeeContractRequest{
			Number:    "PKWT-TEST-001",
			Type:      "PKWT",
			StartDate: "2026-07-29",
			EndDate:   "2027-07-28",
		},
	}

	_, err := service.Create(
		context.Background(),
		domain.Identity{Role: domain.RoleTopManagement},
		request,
		RequestMeta{},
	)
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = service.Create(
		context.Background(),
		domain.Identity{UserID: uuid.New(), Role: domain.RoleHR},
		request,
		RequestMeta{},
	)
	require.NoError(t, err)
	assert.Equal(t, "EMP-001", stub.createCommand.NIP)
	assert.Equal(t, "employee@example.test", stub.createCommand.Email)
	assert.Equal(t, "locked-hash", stub.createCommand.PasswordHash)
}
