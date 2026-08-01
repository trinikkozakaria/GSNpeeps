package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

// ProfileReader membaca detail karyawan milik user yang sedang login.
type ProfileReader interface {
	FindByID(context.Context, uuid.UUID, string) (domain.EmployeeDetail, error)
}

// PersonalMetricsReader menyediakan metrik yang bersumber dari modul Attendance,
// Ketidakhadiran, dan Lembur. Implementasi nyata dipasang pada epic tersebut; sampai saat itu
// adapter sementara mengembalikan nilai nol dan koleksi kosong (keputusan D-020).
type PersonalMetricsReader interface {
	PersonalMetrics(
		context.Context,
		uuid.UUID,
		domain.DashboardRange,
	) (domain.PersonalAttendanceMetrics, error)
}

type ProfileService struct {
	employees ProfileReader
	metrics   PersonalMetricsReader
	now       func() time.Time
}

func NewProfileService(employees ProfileReader, metrics PersonalMetricsReader) *ProfileService {
	return &ProfileService{employees: employees, metrics: metrics, now: time.Now}
}

// Me mengembalikan profil karyawan berdasarkan identity hasil autentikasi, bukan ID dari
// request. Endpoint bersifat read-only dan gaji dibatasi ke periode bulan berjalan.
func (s *ProfileService) Me(
	ctx context.Context,
	identity domain.Identity,
) (domain.EmployeeDetail, error) {
	if identity.EmployeeID == uuid.Nil {
		return domain.EmployeeDetail{}, domain.ErrForbidden
	}
	profile, err := s.employees.FindByID(
		ctx,
		identity.EmployeeID,
		domain.CurrentSalaryPeriod(s.now()),
	)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.EmployeeDetail{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.EmployeeDetail{}, fmt.Errorf("get own profile: %w", err)
	}
	return profile, nil
}

// Metrics mengembalikan metrik personal bulan berjalan. Top Management tidak memiliki
// Metrik Personal dan menerima 403 sesuai kontrak.
func (s *ProfileService) Metrics(
	ctx context.Context,
	identity domain.Identity,
) (domain.PersonalMetrics, error) {
	if identity.Role == domain.RoleTopManagement {
		return domain.PersonalMetrics{}, domain.ErrForbidden
	}
	if identity.EmployeeID == uuid.Nil {
		return domain.PersonalMetrics{}, domain.ErrForbidden
	}
	now := s.now()
	period, err := domain.ResolveDashboardRange(domain.PeriodMonthly, now)
	if err != nil {
		return domain.PersonalMetrics{}, fmt.Errorf("resolve personal metrics period: %w", err)
	}
	attendance, err := s.metrics.PersonalMetrics(ctx, identity.EmployeeID, period)
	if err != nil {
		return domain.PersonalMetrics{}, fmt.Errorf("read personal metrics: %w", err)
	}
	history := attendance.History
	if history == nil {
		history = make([]domain.ClockHistory, 0)
	}
	return domain.PersonalMetrics{
		Period:        domain.CurrentSalaryPeriod(now),
		Present:       attendance.Present,
		Late:          attendance.Late,
		Leave:         attendance.Leave,
		OvertimeHours: attendance.OvertimeHours,
		History:       history,
	}, nil
}
