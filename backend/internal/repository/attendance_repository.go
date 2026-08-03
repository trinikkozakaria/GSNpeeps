package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttendanceRepository struct {
	pool *pgxpool.Pool
}

func NewAttendanceRepository(pool *pgxpool.Pool) *AttendanceRepository {
	return &AttendanceRepository{pool: pool}
}

// FindActiveOfficeLocation mengambil koordinat tepercaya kantor aktif. Koordinat dari client
// tidak pernah dipakai untuk perhitungan radius.
func (r *AttendanceRepository) FindActiveOfficeLocation(
	ctx context.Context,
	id uuid.UUID,
) (domain.OfficeLocation, error) {
	var location domain.OfficeLocation
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT id, kode, nama, alamat, latitude::float8, longitude::float8, is_active
		FROM office_locations
		WHERE id = $1 AND is_active = TRUE
	`, id).Scan(
		&location.ID,
		&location.Code,
		&location.Name,
		&location.Address,
		&location.Latitude,
		&location.Longitude,
		&location.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OfficeLocation{}, ErrNotFound
	}
	if err != nil {
		return domain.OfficeLocation{}, fmt.Errorf("find active office location: %w", err)
	}
	return location, nil
}

func (r *AttendanceRepository) ListActiveOfficeLocations(
	ctx context.Context,
) ([]domain.OfficeLocation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, kode, nama, alamat, latitude::float8, longitude::float8, is_active
		FROM office_locations
		WHERE is_active = TRUE
		ORDER BY nama, id
	`)
	if err != nil {
		return nil, fmt.Errorf("list office locations: %w", err)
	}
	defer rows.Close()

	items := make([]domain.OfficeLocation, 0)
	for rows.Next() {
		var item domain.OfficeLocation
		if err := rows.Scan(
			&item.ID, &item.Code, &item.Name, &item.Address,
			&item.Latitude, &item.Longitude, &item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan office location: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ExistsForDate memeriksa apakah user sudah mencatat tipe absensi tertentu pada satu tanggal.
func (r *AttendanceRepository) ExistsForDate(
	ctx context.Context,
	userID uuid.UUID,
	date string,
	attendanceType domain.AttendanceType,
) (bool, error) {
	var exists bool
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM attendances
			WHERE user_id = $1 AND tanggal = $2::date AND tipe = $3
		)
	`, userID, date, attendanceType).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check attendance existence: %w", err)
	}
	return exists, nil
}

// Create menyimpan satu baris absensi. Unique index (user_id, tanggal, tipe) menjadi penjaga
// terakhir terhadap check-in ganda yang lolos dari pemeriksaan sebelumnya.
func (r *AttendanceRepository) Create(
	ctx context.Context,
	row domain.AttendanceRow,
) (domain.Attendance, error) {
	var record domain.Attendance
	err := executor(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO attendances (
			user_id, tanggal, tipe, mode_kerja, waktu_network, waktu_local,
			gps_lat, gps_long, office_location_id, distance_meters, foto_url, status
		)
		VALUES ($1, $2::date, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, TO_CHAR(tanggal, 'YYYY-MM-DD'), tipe, mode_kerja, waktu_network,
		          gps_lat::float8, gps_long::float8, office_location_id,
		          distance_meters::float8, foto_url, status
	`,
		row.UserID, row.Date, row.Type, row.WorkMode, row.NetworkTime, row.LocalTime,
		row.Latitude, row.Longitude, row.OfficeLocationID, row.DistanceMeters,
		row.PhotoURL, row.Status,
	).Scan(
		&record.ID, &record.Date, &record.Type, &record.WorkMode, &record.Time,
		&record.Latitude, &record.Longitude, &record.OfficeLocationID,
		&record.DistanceMeters, &record.PhotoURL, &record.Status,
	)
	if err != nil {
		return domain.Attendance{}, mapEmployeeMutationError(err)
	}
	return record, nil
}

// LiveFeed mengembalikan absensi seluruh karyawan pada satu tanggal dalam satu query.
func (r *AttendanceRepository) LiveFeed(
	ctx context.Context,
	date string,
) ([]domain.AttendanceLiveFeedItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, e.id, TO_CHAR(a.tanggal, 'YYYY-MM-DD'), a.tipe, a.mode_kerja,
		       a.waktu_network, a.gps_lat::float8, a.gps_long::float8,
		       a.office_location_id, a.distance_meters::float8, a.foto_url, a.status,
		       e.nama, COALESCE(d.nama, '')
		FROM attendances a
		JOIN users u ON u.id = a.user_id
		JOIN employees e ON e.id = u.employee_id
		LEFT JOIN departments d ON d.id = e.department_id
		WHERE a.tanggal = $1::date AND e.deleted_at IS NULL
		ORDER BY a.waktu_network DESC, a.id
	`, date)
	if err != nil {
		return nil, fmt.Errorf("query attendance live feed: %w", err)
	}
	defer rows.Close()

	items := make([]domain.AttendanceLiveFeedItem, 0)
	for rows.Next() {
		var item domain.AttendanceLiveFeedItem
		if err := rows.Scan(
			&item.ID, &item.EmployeeID, &item.Date, &item.Type, &item.WorkMode,
			&item.Time, &item.Latitude, &item.Longitude, &item.OfficeLocationID,
			&item.DistanceMeters, &item.PhotoURL, &item.Status,
			&item.EmployeeName, &item.Department,
		); err != nil {
			return nil, fmt.Errorf("scan attendance live feed: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Report menghitung rekap kehadiran per karyawan dalam satu query agregat. `alpha` adalah
// hari kerja Senin-Jumat dalam rentang yang tidak memiliki check-in valid maupun izin
// yang disetujui.
func (r *AttendanceRepository) Report(
	ctx context.Context,
	filter domain.AttendanceReportFilter,
	workingDays int,
) (domain.AttendanceReportPage, error) {
	start := filter.Start.Format(domain.DateLayout)
	end := filter.End.Format(domain.DateLayout)
	offset := (filter.Page - 1) * filter.Limit

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM employees e
		WHERE e.deleted_at IS NULL AND ($1::uuid IS NULL OR e.department_id = $1)
	`, filter.DepartmentID).Scan(&total); err != nil {
		return domain.AttendanceReportPage{}, fmt.Errorf("count report employees: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		WITH kehadiran AS (
			SELECT u.employee_id,
			       COUNT(*) FILTER (WHERE a.tipe = 'check_in') AS hadir,
			       COUNT(*) FILTER (WHERE a.tipe = 'check_in' AND a.status = 'terlambat') AS terlambat,
			       COALESCE(SUM(
			           EXTRACT(EPOCH FROM (keluar.waktu_network - a.waktu_network)) / 3600
			       ) FILTER (WHERE keluar.waktu_network IS NOT NULL), 0) AS total_jam
			FROM attendances a
			JOIN users u ON u.id = a.user_id
			LEFT JOIN attendances keluar
			       ON keluar.user_id = a.user_id
			      AND keluar.tanggal = a.tanggal
			      AND keluar.tipe = 'check_out'
			WHERE a.tanggal BETWEEN $1::date AND $2::date AND a.tipe = 'check_in'
			GROUP BY u.employee_id
		),
		izin AS (
			SELECT u.employee_id,
			       COALESCE(SUM(
			           (LEAST(lr.tanggal_selesai, $2::date) - GREATEST(lr.tanggal_mulai, $1::date)) + 1
			       ), 0) AS hari_izin
			FROM leave_requests lr
			JOIN users u ON u.id = lr.user_id
			WHERE lr.status = 'disetujui'
			  AND lr.tanggal_mulai <= $2::date
			  AND lr.tanggal_selesai >= $1::date
			GROUP BY u.employee_id
		)
		SELECT e.id, e.nama, COALESCE(d.nama, ''),
		       COALESCE(kehadiran.hadir, 0),
		       COALESCE(kehadiran.terlambat, 0),
		       COALESCE(izin.hari_izin, 0),
		       COALESCE(kehadiran.total_jam, 0)::float8
		FROM employees e
		LEFT JOIN departments d ON d.id = e.department_id
		LEFT JOIN kehadiran ON kehadiran.employee_id = e.id
		LEFT JOIN izin ON izin.employee_id = e.id
		WHERE e.deleted_at IS NULL AND ($3::uuid IS NULL OR e.department_id = $3)
		ORDER BY e.nama, e.id
		LIMIT $4 OFFSET $5
	`, start, end, filter.DepartmentID, filter.Limit, offset)
	if err != nil {
		return domain.AttendanceReportPage{}, fmt.Errorf("query attendance report: %w", err)
	}
	defer rows.Close()

	items := make([]domain.AttendanceReportItem, 0)
	for rows.Next() {
		var item domain.AttendanceReportItem
		var leaveDays int
		if err := rows.Scan(
			&item.EmployeeID, &item.EmployeeName, &item.Department,
			&item.Present, &item.Late, &leaveDays, &item.TotalHours,
		); err != nil {
			return domain.AttendanceReportPage{}, fmt.Errorf("scan attendance report: %w", err)
		}
		item.Leave = leaveDays
		item.Absent = workingDays - item.Present - item.Leave
		if item.Absent < 0 {
			item.Absent = 0
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.AttendanceReportPage{}, fmt.Errorf("iterate attendance report: %w", err)
	}
	return domain.AttendanceReportPage{
		Items: items, Total: total, Page: filter.Page, Limit: filter.Limit,
	}, nil
}

// PersonalMetrics memenuhi kontrak metrik personal memakai data absensi nyata.
func (r *AttendanceRepository) PersonalMetrics(
	ctx context.Context,
	employeeID uuid.UUID,
	period domain.DashboardRange,
) (domain.PersonalAttendanceMetrics, error) {
	start := period.Start.Format(domain.DateLayout)
	end := period.End.Format(domain.DateLayout)

	var metrics domain.PersonalAttendanceMetrics
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE a.tipe = 'check_in'),
			COUNT(*) FILTER (WHERE a.tipe = 'check_in' AND a.status = 'terlambat')
		FROM attendances a
		JOIN users u ON u.id = a.user_id
		WHERE u.employee_id = $1 AND a.tanggal BETWEEN $2::date AND $3::date
	`, employeeID, start, end).Scan(&metrics.Present, &metrics.Late)
	if err != nil {
		return domain.PersonalAttendanceMetrics{}, fmt.Errorf("aggregate personal attendance: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(
		           (LEAST(lr.tanggal_selesai, $3::date) - GREATEST(lr.tanggal_mulai, $2::date)) + 1
		       ), 0)
		FROM leave_requests lr
		JOIN users u ON u.id = lr.user_id
		WHERE u.employee_id = $1 AND lr.status = 'disetujui'
		  AND lr.tanggal_mulai <= $3::date AND lr.tanggal_selesai >= $2::date
	`, employeeID, start, end).Scan(&metrics.Leave)
	if err != nil {
		return domain.PersonalAttendanceMetrics{}, fmt.Errorf("aggregate personal leave: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(o.durasi_jam), 0)::float8
		FROM overtime_requests o
		JOIN users u ON u.id = o.user_id
		WHERE u.employee_id = $1 AND o.status = 'disetujui'
		  AND o.tanggal BETWEEN $2::date AND $3::date
	`, employeeID, start, end).Scan(&metrics.OvertimeHours)
	if err != nil {
		return domain.PersonalAttendanceMetrics{}, fmt.Errorf("aggregate personal overtime: %w", err)
	}

	history, err := r.clockHistory(ctx, employeeID, start, end)
	if err != nil {
		return domain.PersonalAttendanceMetrics{}, err
	}
	metrics.History = history
	return metrics, nil
}

func (r *AttendanceRepository) clockHistory(
	ctx context.Context,
	employeeID uuid.UUID,
	start, end string,
) ([]domain.ClockHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT TO_CHAR(a.tanggal, 'YYYY-MM-DD'),
		       MAX(CASE WHEN a.tipe = 'check_in' THEN TO_CHAR(a.waktu_local, 'HH24:MI') END),
		       MAX(CASE WHEN a.tipe = 'check_out' THEN TO_CHAR(a.waktu_local, 'HH24:MI') END),
		       MAX(CASE WHEN a.tipe = 'check_in' THEN a.status END)
		FROM attendances a
		JOIN users u ON u.id = a.user_id
		WHERE u.employee_id = $1 AND a.tanggal BETWEEN $2::date AND $3::date
		GROUP BY a.tanggal
		ORDER BY a.tanggal DESC
	`, employeeID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query clock history: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ClockHistory, 0)
	for rows.Next() {
		var item domain.ClockHistory
		var status *string
		if err := rows.Scan(&item.Date, &item.CheckIn, &item.CheckOut, &status); err != nil {
			return nil, fmt.Errorf("scan clock history: %w", err)
		}
		if status != nil {
			item.Status = *status
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// AttendanceMetrics memenuhi kontrak metrik dashboard memakai data absensi nyata.
func (r *AttendanceRepository) AttendanceMetrics(
	ctx context.Context,
	period domain.DashboardRange,
) (domain.AttendanceMetrics, error) {
	var metrics domain.AttendanceMetrics
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(DISTINCT (user_id, tanggal)) FILTER (WHERE tipe = 'check_in'),
			COUNT(DISTINCT (user_id, tanggal)) FILTER (WHERE tipe = 'check_in' AND status = 'terlambat')
		FROM attendances
		WHERE tanggal BETWEEN $1::date AND $2::date
	`, period.Start.Format(domain.DateLayout), period.End.Format(domain.DateLayout),
	).Scan(&metrics.ValidAttendance, &metrics.Late)
	if err != nil {
		return domain.AttendanceMetrics{}, fmt.Errorf("aggregate attendance metrics: %w", err)
	}
	return metrics, nil
}

// ClaimExpiredPhotos mengambil sejumlah foto absensi yang melewati masa retensi. Baris
// dikunci dengan SKIP LOCKED agar aman dijalankan beberapa worker sekaligus.
func (r *AttendanceRepository) ClaimExpiredPhotos(
	ctx context.Context,
	before time.Time,
	limit int,
) ([]domain.ExpiredPhoto, error) {
	rows, err := executor(ctx, r.pool).Query(ctx, `
		SELECT id, foto_url
		FROM attendances
		WHERE foto_url IS NOT NULL AND tanggal < $1::date
		ORDER BY tanggal
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, before.Format(domain.DateLayout), limit)
	if err != nil {
		return nil, fmt.Errorf("claim expired attendance photos: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ExpiredPhoto, 0)
	for rows.Next() {
		var item domain.ExpiredPhoto
		if err := rows.Scan(&item.AttendanceID, &item.PhotoURL); err != nil {
			return nil, fmt.Errorf("scan expired attendance photo: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ClearPhotoURL dipanggil hanya setelah berkas berhasil dihapus dari storage. Baris absensi
// tidak pernah dihapus.
func (r *AttendanceRepository) ClearPhotoURL(ctx context.Context, attendanceID uuid.UUID) error {
	_, err := executor(ctx, r.pool).Exec(ctx, `
		UPDATE attendances SET foto_url = NULL WHERE id = $1
	`, attendanceID)
	if err != nil {
		return fmt.Errorf("clear attendance photo url: %w", err)
	}
	return nil
}
