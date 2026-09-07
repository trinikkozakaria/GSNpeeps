// Integration test yang membuktikan riwayat check-in/check-out pada Metrik Personal
// (GET /profil/saya/metrik) memakai jam WIB, bukan jam UTC session Postgres. Dilewati
// kecuali TEST_DATABASE_URL menunjuk ke PostgreSQL bermigrasi.
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

func TestPersonalMetricsClockHistoryUsesJakartaTime(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL tidak diset; integration test dilewati")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))
	defer pool.Close()

	// Sesi Postgres di lingkungan uji ini defaultnya UTC (tidak ada TZ/timezone di-set),
	// persis seperti container aplikasi. Ini membuktikan bug tidak lolos karena kebetulan
	// session sudah WIB.
	var sessionTimezone string
	require.NoError(t, pool.QueryRow(ctx, `SHOW timezone`).Scan(&sessionTimezone))
	require.Equal(t, "UTC", sessionTimezone, "test ini mengasumsikan session default UTC")

	suffix := uuid.NewString()[:8]
	var departmentID, positionID, employeeID, userID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO departments (nama) VALUES ($1) RETURNING id`, "Uji Metrik "+suffix,
	).Scan(&departmentID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO positions (nama, department_id) VALUES ($1, $2) RETURNING id`,
		"Uji Posisi "+suffix, departmentID,
	).Scan(&positionID))
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO employees (nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join, department_id, position_id, status)
		VALUES ($1, $2, 'P', '1995-01-01', '2026-01-05', $3, $4, 'aktif')
		RETURNING id
	`, "UJI-METRIK-"+suffix, "Karyawan Metrik "+suffix, departmentID, positionID).Scan(&employeeID))
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO users (employee_id, email, password_hash, role_id)
		VALUES ($1, $2, 'x', (SELECT id FROM roles WHERE nama = 'karyawan'))
		RETURNING id
	`, employeeID, "karyawan-metrik-"+suffix+"@example.test").Scan(&userID))

	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM attendances WHERE user_id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM employees WHERE id = $1`, employeeID)
		_, _ = pool.Exec(c, `DELETE FROM positions WHERE id = $1`, positionID)
		_, _ = pool.Exec(c, `DELETE FROM departments WHERE id = $1`, departmentID)
	})

	// Check-in jam 09:15 WIB (= 02:15 UTC), persis seperti attendance_service.go menyimpan
	// waktu_local sebagai instant hasil localTime.In(domain.Jakarta()).
	checkIn := time.Date(2026, 8, 14, 9, 15, 0, 0, domain.Jakarta())
	checkOut := time.Date(2026, 8, 14, 18, 5, 0, 0, domain.Jakarta())
	_, err = pool.Exec(ctx, `
		INSERT INTO attendances (user_id, tanggal, tipe, mode_kerja, waktu_network, waktu_local, gps_lat, gps_long, status)
		VALUES ($1, '2026-08-14', 'check_in', 'WFH', $2, $2, -6.2, 106.8, 'tepat_waktu')
	`, userID, checkIn)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO attendances (user_id, tanggal, tipe, mode_kerja, waktu_network, waktu_local, gps_lat, gps_long, status)
		VALUES ($1, '2026-08-14', 'check_out', 'WFH', $2, $2, -6.2, 106.8, 'valid')
	`, userID, checkOut)
	require.NoError(t, err)

	repo := repository.NewAttendanceRepository(pool)
	metrics, err := repo.PersonalMetrics(ctx, employeeID, domain.DashboardRange{
		Start: time.Date(2026, 8, 1, 0, 0, 0, 0, domain.Jakarta()),
		End:   time.Date(2026, 8, 31, 0, 0, 0, 0, domain.Jakarta()),
	})
	require.NoError(t, err)
	require.Len(t, metrics.History, 1)

	entry := metrics.History[0]
	require.NotNil(t, entry.CheckIn)
	require.NotNil(t, entry.CheckOut)
	assert.Equal(t, "09:15", *entry.CheckIn, "check-in harus WIB (09:15), bukan UTC (02:15)")
	assert.Equal(t, "18:05", *entry.CheckOut, "check-out harus WIB (18:05), bukan UTC (11:05)")
}
