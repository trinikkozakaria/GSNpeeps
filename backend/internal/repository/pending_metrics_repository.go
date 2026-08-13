package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// PendingMetricsRepository memenuhi kontrak metrik yang bersumber dari tabel `attendances`,
// `leave_requests`, dan `overtime_requests`. Tabel tersebut dibuat pada epic Attendance dan
// Approval, sehingga adapter ini mengembalikan nilai nol dan koleksi kosong.
//
// Nilai nol adalah empty state yang sah menurut schema (minimum 0 dan array boleh kosong);
// tidak ada data yang dikarang. Ganti adapter ini dengan repository nyata saat tabel sumber
// tersedia — response schema tidak perlu berubah. Lihat keputusan D-020.
type PendingMetricsRepository struct{}

func NewPendingMetricsRepository() *PendingMetricsRepository {
	return &PendingMetricsRepository{}
}

func (r *PendingMetricsRepository) AttendanceMetrics(
	context.Context,
	domain.DashboardRange,
) (domain.AttendanceMetrics, error) {
	return domain.AttendanceMetrics{}, nil
}

func (r *PendingMetricsRepository) LeaveMetrics(
	context.Context,
	domain.DashboardRange,
) (domain.LeaveMetrics, error) {
	return domain.LeaveMetrics{}, nil
}

func (r *PendingMetricsRepository) PersonalMetrics(
	context.Context,
	uuid.UUID,
	domain.DashboardRange,
) (domain.PersonalAttendanceMetrics, error) {
	return domain.PersonalAttendanceMetrics{
		History: make([]domain.ClockHistory, 0),
	}, nil
}
