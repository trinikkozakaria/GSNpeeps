// Package tests berisi integration test yang membutuhkan PostgreSQL nyata.
//
// Jalankan dengan database yang sudah dimigrasi:
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5432/gsnpeeps_test?sslmode=disable \
//	    go test ./tests/...
//
// Tanpa environment tersebut test di-skip agar `go test ./...` tetap dapat berjalan pada
// mesin yang tidak menyediakan database. Seluruh data yang dibuat bersifat sintetis dan
// dihapus kembali; tidak ada data karyawan nyata yang dipakai.
package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	pool          *pgxpool.Pool
	employees     *repository.EmployeeRepository
	dashboard     *repository.DashboardRepository
	departmentID  uuid.UUID
	positionID    uuid.UUID
	activeID      uuid.UUID
	inactiveID    uuid.UUID
	subordinateID uuid.UUID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL tidak diset; integration test dilewati")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))

	suffix := uuid.NewString()[:8]
	f := &fixture{
		pool:      pool,
		employees: repository.NewEmployeeRepository(pool),
		dashboard: repository.NewDashboardRepository(pool),
	}

	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO departments (nama) VALUES ($1) RETURNING id`,
		"Uji Teknologi "+suffix,
	).Scan(&f.departmentID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO positions (nama, department_id) VALUES ($1, $2) RETURNING id`,
		"Uji Staff "+suffix, f.departmentID,
	).Scan(&f.positionID))

	f.activeID = f.insertEmployee(t, "UJI-"+suffix+"-1", "Anita Sintetis", "P", "2026-01-05", nil)
	f.subordinateID = f.insertEmployee(t, "UJI-"+suffix+"-2", "Budi Sintetis", "L", "2026-02-10", &f.activeID)
	f.inactiveID = f.insertEmployee(t, "UJI-"+suffix+"-3", "Citra Sintetis", "P", "2026-01-20", nil)
	_, err = pool.Exec(ctx,
		`UPDATE employees SET status = 'nonaktif', deleted_at = $2 WHERE id = $1`,
		f.inactiveID, time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		ids := []uuid.UUID{f.subordinateID, f.activeID, f.inactiveID}
		for _, table := range []string{
			"employee_documents", "employee_salaries", "employee_position_history",
			"employee_education", "employee_emergency_contacts", "employee_npwp",
			"employee_bpjs", "employee_contracts", "employee_ktp", "employee_addresses",
		} {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM `+table+` WHERE employee_id = ANY($1)`, ids)
		}
		_, _ = pool.Exec(cleanupCtx, `UPDATE employees SET atasan_id = NULL WHERE id = ANY($1)`, ids)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM employees WHERE id = ANY($1)`, ids)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM positions WHERE id = $1`, f.positionID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM departments WHERE id = $1`, f.departmentID)
		pool.Close()
	})
	return f
}

func (f *fixture) insertEmployee(
	t *testing.T,
	nip, name, gender, joinDate string,
	supervisor *uuid.UUID,
) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(context.Background(), `
		INSERT INTO employees (
			nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join,
			department_id, position_id, atasan_id, status
		)
		VALUES ($1, $2, $3, '1995-01-01', $4, $5, $6, $7, 'aktif')
		RETURNING id
	`, nip, name, gender, joinDate, f.departmentID, f.positionID, supervisor).Scan(&id))
	return id
}

// Skema hasil migration harus memakai nama tabel Database Schema v1.1.
func TestMigrationCreatesApprovedEmployeeTables(t *testing.T) {
	f := newFixture(t)

	expected := []string{
		"employee_addresses", "employee_ktp", "employee_contracts", "employee_bpjs",
		"employee_npwp", "employee_emergency_contacts", "employee_education",
		"employee_position_history", "employee_salaries", "employee_documents",
	}
	for _, table := range expected {
		var exists bool
		require.NoError(t, f.pool.QueryRow(context.Background(), `
			SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists))
		assert.Truef(t, exists, "tabel %s harus ada setelah migration", table)
	}
}

func TestEmployeeSalaryIsUniquePerPeriod(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	_, err := f.pool.Exec(ctx, `
		INSERT INTO employee_salaries (employee_id, periode, gaji_pokok, tunjangan, potongan)
		VALUES ($1, '2026-08', 10000000, 2000000, 500000)
	`, f.activeID)
	require.NoError(t, err)

	_, err = f.pool.Exec(ctx, `
		INSERT INTO employee_salaries (employee_id, periode, gaji_pokok)
		VALUES ($1, '2026-08', 9000000)
	`, f.activeID)
	require.Error(t, err, "periode gaji harus unik per karyawan")

	// take_home_pay adalah kolom generated sehingga tidak dapat menyimpang dari komponennya.
	var takeHomePay float64
	require.NoError(t, f.pool.QueryRow(ctx, `
		SELECT take_home_pay::float8 FROM employee_salaries
		WHERE employee_id = $1 AND periode = '2026-08'
	`, f.activeID).Scan(&takeHomePay))
	assert.InDelta(t, 11500000, takeHomePay, 0.01)
}

func TestEmployeeListFiltersSearchAndPagination(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	page, err := f.employees.List(ctx, domain.EmployeeFilter{
		Search:       "Anita Sintetis",
		DepartmentID: &f.departmentID,
		Page:         1,
		Limit:        10,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, f.activeID, page.Items[0].ID)

	// Pencarian dibatasi pada nama dan NIP; email bukan field pencarian kontrak.
	page, err = f.employees.List(ctx, domain.EmployeeFilter{
		Search:       "example.test",
		DepartmentID: &f.departmentID,
		Page:         1,
		Limit:        10,
	})
	require.NoError(t, err)
	assert.Empty(t, page.Items)

	page, err = f.employees.List(ctx, domain.EmployeeFilter{
		DepartmentID: &f.departmentID,
		Status:       "nonaktif",
		Page:         1,
		Limit:        10,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, f.inactiveID, page.Items[0].ID)

	first, err := f.employees.List(ctx, domain.EmployeeFilter{
		DepartmentID: &f.departmentID, Page: 1, Limit: 2,
	})
	require.NoError(t, err)
	second, err := f.employees.List(ctx, domain.EmployeeFilter{
		DepartmentID: &f.departmentID, Page: 2, Limit: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, first.Total)
	assert.Len(t, first.Items, 2)
	assert.Len(t, second.Items, 1)
	assert.NotEqual(t, first.Items[0].ID, second.Items[0].ID)
}

func TestEmployeeDetailLoadsNestedSectionsAndCurrentMonthSalary(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	_, err := f.pool.Exec(ctx, `
		INSERT INTO employee_addresses (employee_id, jalan, kota, provinsi)
		VALUES ($1, 'Jalan Sintetis 1', 'Jakarta', 'DKI Jakarta')
	`, f.activeID)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `
		INSERT INTO employee_bpjs (employee_id, no_bpjs_kesehatan, no_bpjs_ketenagakerjaan)
		VALUES ($1, '0001112223', '0004445556')
	`, f.activeID)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `
		INSERT INTO employee_emergency_contacts (employee_id, nama, hubungan, no_hp)
		VALUES ($1, 'Kontak Sintetis', 'Saudara', '08000000000')
	`, f.activeID)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `
		INSERT INTO employee_education (employee_id, jenjang, institusi, tahun_lulus)
		VALUES ($1, 'S1', 'Universitas Sintetis', 2017)
	`, f.activeID)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `
		INSERT INTO employee_position_history (
			employee_id, jabatan, tanggal_mulai, department_id, position_id
		)
		VALUES ($1, 'Staff', '2026-01-05', $2, $3)
	`, f.activeID, f.departmentID, f.positionID)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `
		INSERT INTO employee_salaries (employee_id, periode, gaji_pokok)
		VALUES ($1, '2026-07', 8000000), ($1, '2026-08', 9000000)
	`, f.activeID)
	require.NoError(t, err)

	detail, err := f.employees.FindByID(ctx, f.activeID, "2026-08")
	require.NoError(t, err)

	require.NotNil(t, detail.Address)
	assert.Equal(t, "Jakarta", detail.Address.City)
	assert.Len(t, detail.BPJS, 2)
	require.Len(t, detail.EmergencyContacts, 1)
	assert.Equal(t, "Kontak Sintetis", detail.EmergencyContacts[0].Name)
	require.Len(t, detail.Education, 1)
	require.NotNil(t, detail.Education[0].GraduationYear)
	assert.Equal(t, 2017, *detail.Education[0].GraduationYear)
	require.Len(t, detail.PositionHistory, 1)
	require.NotNil(t, detail.PositionHistory[0].Position)
	assert.Equal(t, f.positionID, detail.PositionHistory[0].Position.ID)

	// Hanya gaji periode yang diminta yang boleh muncul; histori tidak pernah dikirim.
	require.NotNil(t, detail.CurrentSalary)
	assert.Equal(t, "2026-08", detail.CurrentSalary.Period)
	assert.InDelta(t, 9000000, detail.CurrentSalary.BasePay, 0.01)
}

func TestEmployeeDocumentsStoreLocatorOnly(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	documentID, err := f.employees.CreateDocument(ctx, domain.NewEmployeeDocument{
		EmployeeID: f.activeID,
		Type:       "Ijazah",
		FileName:   "ijazah.pdf",
		FileURL:    "GSNpeeps/employee-documents/" + f.activeID.String() + "/berkas.pdf",
	})
	require.NoError(t, err)

	documents, err := f.employees.FindDocuments(ctx, f.activeID)
	require.NoError(t, err)
	require.Len(t, documents, 1)
	assert.Equal(t, documentID, documents[0].ID)
	assert.Equal(t, "ijazah.pdf", documents[0].FileName)
	assert.False(t, documents[0].CreatedAt.IsZero())

	// Karyawan yang sudah soft-delete tidak dapat menerima dokumen baru.
	require.ErrorIs(t, f.employees.ExistsActive(ctx, f.inactiveID), repository.ErrNotFound)
	require.NoError(t, f.employees.ExistsActive(ctx, f.activeID))
}

func TestEmployeeExportRowsHonourFilters(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	rows, err := f.employees.ExportRows(ctx, domain.EmployeeExportQuery{
		Filter: domain.EmployeeFilter{DepartmentID: &f.departmentID},
	}, 5000)
	require.NoError(t, err)
	assert.Len(t, rows, 3)

	rows, err = f.employees.ExportRows(ctx, domain.EmployeeExportQuery{
		EmployeeID: &f.activeID,
	}, 5000)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, f.activeID, rows[0].ID)

	rows, err = f.employees.ExportRows(ctx, domain.EmployeeExportQuery{
		Filter: domain.EmployeeFilter{
			DepartmentID: &f.departmentID,
			Search:       "tidak-ada-kecocokan",
		},
	}, 5000)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestDashboardSnapshotSeparatesActiveAndInactive(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	period, err := domain.ResolveDashboardRange(
		domain.PeriodMonthly,
		time.Date(2026, time.August, 15, 0, 0, 0, 0, domain.Jakarta()),
	)
	require.NoError(t, err)

	snapshot, err := f.dashboard.Snapshot(ctx, period)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, snapshot.HeadcountStart, 3)
	assert.GreaterOrEqual(t, snapshot.Resigned, 1, "deleted_at 10 Agustus berada dalam periode")
	assert.NotEmpty(t, snapshot.ActiveDepartments)
	assert.NotEmpty(t, snapshot.InactiveDepartments)

	total := 0
	for _, gender := range snapshot.GenderRatio {
		total += gender.Count
	}
	assert.Equal(t, snapshot.ActiveEmployees, total,
		"rasio gender harus mencakup seluruh populasi aktif")

	found := false
	for _, member := range snapshot.OrganizationMembers {
		if member.EmployeeID == f.subordinateID {
			require.NotNil(t, member.SupervisorID)
			assert.Equal(t, f.activeID, *member.SupervisorID)
			found = true
		}
	}
	assert.True(t, found, "bawahan langsung harus muncul pada org chart")
}

func TestDashboardSalaryTotalsAggregateByPeriod(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	_, err := f.pool.Exec(ctx, `
		INSERT INTO employee_salaries (employee_id, periode, gaji_pokok, tunjangan)
		VALUES ($1, '2026-08', 10000000, 1000000), ($2, '2026-08', 5000000, 0)
	`, f.activeID, f.subordinateID)
	require.NoError(t, err)

	totals, err := f.dashboard.SalaryTotals(ctx, []string{"2026-08", "2026-09"})
	require.NoError(t, err)

	assert.GreaterOrEqual(t, totals["2026-08"], 16000000.0)
	assert.NotContains(t, totals, "2026-09", "periode tanpa data tidak menghasilkan entry")
}
