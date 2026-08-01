package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type profileReaderStub struct {
	requestedID     uuid.UUID
	requestedPeriod string
	detail          domain.EmployeeDetail
	err             error
}

func (s *profileReaderStub) FindByID(
	_ context.Context,
	id uuid.UUID,
	salaryPeriod string,
) (domain.EmployeeDetail, error) {
	s.requestedID = id
	s.requestedPeriod = salaryPeriod
	return s.detail, s.err
}

func newProfileServiceForTest(reader ProfileReader) *ProfileService {
	service := NewProfileService(reader, repository.NewPendingMetricsRepository())
	service.now = func() time.Time {
		return time.Date(2026, time.August, 1, 9, 30, 0, 0, domain.Jakarta())
	}
	return service
}

func TestProfileResolvesEmployeeFromIdentityNotRequest(t *testing.T) {
	stub := &profileReaderStub{}
	service := newProfileServiceForTest(stub)
	employeeID := uuid.New()

	_, err := service.Me(context.Background(), domain.Identity{
		UserID:     uuid.New(),
		EmployeeID: employeeID,
		Role:       domain.RoleEmployee,
	})

	require.NoError(t, err)
	assert.Equal(t, employeeID, stub.requestedID)
}

func TestProfileLimitsSalaryToCurrentMonth(t *testing.T) {
	stub := &profileReaderStub{}
	service := newProfileServiceForTest(stub)

	_, err := service.Me(context.Background(), domain.Identity{
		EmployeeID: uuid.New(),
		Role:       domain.RoleHR,
	})

	require.NoError(t, err)
	assert.Equal(t, "2026-08", stub.requestedPeriod)
}

func TestProfileRejectsIdentityWithoutEmployee(t *testing.T) {
	service := newProfileServiceForTest(&profileReaderStub{})

	_, err := service.Me(context.Background(), domain.Identity{Role: domain.RoleTopManagement})

	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestPersonalMetricsForbiddenForTopManagement(t *testing.T) {
	service := newProfileServiceForTest(&profileReaderStub{})

	_, err := service.Metrics(context.Background(), domain.Identity{
		EmployeeID: uuid.New(),
		Role:       domain.RoleTopManagement,
	})

	require.ErrorIs(t, err, domain.ErrForbidden)
}

// Sampai modul Attendance tersedia, metrik personal mengembalikan empty state yang sah
// menurut kontrak dan tidak boleh berisi data karangan (D-020).
func TestPersonalMetricsReturnsContractEmptyStateWhileAttendancePending(t *testing.T) {
	service := newProfileServiceForTest(&profileReaderStub{})

	metrics, err := service.Metrics(context.Background(), domain.Identity{
		EmployeeID: uuid.New(),
		Role:       domain.RoleEmployee,
	})

	require.NoError(t, err)
	assert.Equal(t, "2026-08", metrics.Period)
	assert.Zero(t, metrics.Present)
	assert.Zero(t, metrics.Late)
	assert.Zero(t, metrics.Leave)
	assert.Zero(t, metrics.OvertimeHours)
	assert.NotNil(t, metrics.History)
	assert.Empty(t, metrics.History)
}
