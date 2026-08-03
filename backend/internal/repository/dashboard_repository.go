package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardRepository membaca agregasi employee-side untuk dashboard HR. Seluruh query
// bersifat agregat sehingga tidak ada round trip per karyawan.
type DashboardRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

// Snapshot menghitung headcount, join, resign, komposisi departemen, rasio gender, dan
// anggota org chart untuk satu rentang periode.
//
// Status aktif pada suatu titik waktu T diturunkan dari timeline: karyawan dianggap aktif
// bila tanggal_join <= T dan belum dinonaktifkan pada T (deleted_at NULL atau > T). Pemetaan
// cuti/resign mengikuti keputusan D-019.
func (r *DashboardRepository) Snapshot(
	ctx context.Context,
	period domain.DashboardRange,
) (domain.DashboardSnapshot, error) {
	start := period.Start
	end := period.ExclusiveEnd()

	var snapshot domain.DashboardSnapshot
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (
				WHERE tanggal_join < $1::date
				  AND (deleted_at IS NULL OR deleted_at >= $1::timestamp)
			) AS headcount_start,
			COUNT(*) FILTER (
				WHERE tanggal_join < $2::date
				  AND (deleted_at IS NULL OR deleted_at >= $2::timestamp)
			) AS active_end,
			COUNT(*) FILTER (
				WHERE tanggal_join < $2::date
				  AND deleted_at IS NOT NULL AND deleted_at < $2::timestamp
			) AS inactive_end,
			COUNT(*) FILTER (
				WHERE tanggal_join >= $1::date AND tanggal_join < $2::date
			) AS new_employees,
			COUNT(*) FILTER (
				WHERE deleted_at >= $1::timestamp AND deleted_at < $2::timestamp
			) AS resigned
		FROM employees
	`, start, end).Scan(
		&snapshot.HeadcountStart,
		&snapshot.ActiveEmployees,
		&snapshot.InactiveEmployees,
		&snapshot.NewEmployees,
		&snapshot.Resigned,
	)
	if err != nil {
		return domain.DashboardSnapshot{}, fmt.Errorf("aggregate employee headcount: %w", err)
	}

	if snapshot.ActiveDepartments, snapshot.InactiveDepartments, err = r.departmentComposition(ctx, end); err != nil {
		return domain.DashboardSnapshot{}, err
	}
	if snapshot.GenderRatio, err = r.genderRatio(ctx, end); err != nil {
		return domain.DashboardSnapshot{}, err
	}
	if snapshot.OrganizationMembers, err = r.organizationMembers(ctx, end); err != nil {
		return domain.DashboardSnapshot{}, err
	}
	return snapshot, nil
}

func (r *DashboardRepository) departmentComposition(
	ctx context.Context,
	end time.Time,
) ([]domain.NamedCount, []domain.NamedCount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(d.nama, 'Tanpa Departemen') AS departemen,
		       COUNT(*) FILTER (WHERE e.deleted_at IS NULL OR e.deleted_at >= $1::timestamp) AS aktif,
		       COUNT(*) FILTER (WHERE e.deleted_at IS NOT NULL AND e.deleted_at < $1::timestamp) AS nonaktif
		FROM employees e
		LEFT JOIN departments d ON d.id = e.department_id
		WHERE e.tanggal_join < $1::date
		GROUP BY departemen
		ORDER BY departemen
	`, end)
	if err != nil {
		return nil, nil, fmt.Errorf("aggregate department composition: %w", err)
	}
	defer rows.Close()

	active := make([]domain.NamedCount, 0)
	inactive := make([]domain.NamedCount, 0)
	for rows.Next() {
		var name string
		var activeCount, inactiveCount int
		if err := rows.Scan(&name, &activeCount, &inactiveCount); err != nil {
			return nil, nil, fmt.Errorf("scan department composition: %w", err)
		}
		if activeCount > 0 {
			active = append(active, domain.NamedCount{Name: name, Count: activeCount})
		}
		if inactiveCount > 0 {
			inactive = append(inactive, domain.NamedCount{Name: name, Count: inactiveCount})
		}
	}
	return active, inactive, rows.Err()
}

// genderRatio hanya menghitung populasi aktif pada akhir periode. Gender yang belum diisi
// masuk kategori belum_diisi dan tidak pernah digabung ke laki_laki/perempuan (D-015).
func (r *DashboardRepository) genderRatio(
	ctx context.Context,
	end time.Time,
) ([]domain.GenderCount, error) {
	var male, female, unassigned int
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE jenis_kelamin = 'L'),
			COUNT(*) FILTER (WHERE jenis_kelamin = 'P'),
			COUNT(*) FILTER (WHERE jenis_kelamin IS NULL OR jenis_kelamin NOT IN ('L', 'P'))
		FROM employees
		WHERE tanggal_join < $1::date
		  AND (deleted_at IS NULL OR deleted_at >= $1::timestamp)
	`, end).Scan(&male, &female, &unassigned)
	if err != nil {
		return nil, fmt.Errorf("aggregate gender ratio: %w", err)
	}
	return []domain.GenderCount{
		{Category: domain.GenderMale, Count: male},
		{Category: domain.GenderFemale, Count: female},
		{Category: domain.GenderUnassigned, Count: unassigned},
	}, nil
}

func (r *DashboardRepository) organizationMembers(
	ctx context.Context,
	end time.Time,
) ([]domain.OrganizationMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.atasan_id, e.nama,
		       COALESCE(d.nama, ''), COALESCE(p.nama, '')
		FROM employees e
		LEFT JOIN departments d ON d.id = e.department_id
		LEFT JOIN positions p ON p.id = e.position_id
		WHERE e.tanggal_join < $1::date
		  AND (e.deleted_at IS NULL OR e.deleted_at >= $1::timestamp)
		ORDER BY e.nama, e.id
	`, end)
	if err != nil {
		return nil, fmt.Errorf("query organization members: %w", err)
	}
	defer rows.Close()

	members := make([]domain.OrganizationMember, 0)
	for rows.Next() {
		var member domain.OrganizationMember
		if err := rows.Scan(
			&member.EmployeeID,
			&member.SupervisorID,
			&member.Name,
			&member.Department,
			&member.Position,
		); err != nil {
			return nil, fmt.Errorf("scan organization member: %w", err)
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

// SalaryTotals mengembalikan total take_home_pay per periode YYYY-MM untuk periode yang
// diminta. Alokasi proporsional terhadap hari kerja dihitung di service (D-015).
func (r *DashboardRepository) SalaryTotals(
	ctx context.Context,
	periods []string,
) (map[string]float64, error) {
	totals := make(map[string]float64, len(periods))
	if len(periods) == 0 {
		return totals, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT periode, COALESCE(SUM(take_home_pay), 0)::float8
		FROM employee_salaries
		WHERE periode = ANY($1::char(7)[])
		GROUP BY periode
	`, periods)
	if err != nil {
		return nil, fmt.Errorf("aggregate payroll totals: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var period string
		var total float64
		if err := rows.Scan(&period, &total); err != nil {
			return nil, fmt.Errorf("scan payroll total: %w", err)
		}
		totals[period] = total
	}
	return totals, rows.Err()
}
