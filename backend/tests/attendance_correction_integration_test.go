// Integration test untuk koreksi absensi: histori penuh per role (Atasan/HR) dan tahap
// Top Management atas pengajuan HR. Menjalankan UATHandler langsung dengan identity
// disuntik ke context, sama seperti document_type_integration_test.go, karena UATHandler
// memakai *pgxpool.Pool mentah. Dilewati kecuali TEST_DATABASE_URL menunjuk ke PostgreSQL
// bermigrasi (migration 00021 wajib sudah jalan).
package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/handler"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type correctionFixture struct {
	pool    *pgxpool.Pool
	handler *handler.UATHandler

	supervisorEmployee, subordinateEmployee uuid.UUID
	hrEmployee, tmEmployee                  uuid.UUID
	supervisorUser, subordinateUser         uuid.UUID
	hrUser, tmUser                          uuid.UUID
}

func newCorrectionFixture(t *testing.T) *correctionFixture {
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
	f := &correctionFixture{pool: pool, handler: handler.NewUATHandler(pool)}

	var departmentID, positionID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO departments (nama) VALUES ($1) RETURNING id`, "Uji Koreksi "+suffix,
	).Scan(&departmentID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO positions (nama, department_id) VALUES ($1, $2) RETURNING id`,
		"Uji Posisi "+suffix, departmentID,
	).Scan(&positionID))

	insertEmployee := func(nip, name string, supervisor *uuid.UUID) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `
			INSERT INTO employees (nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join, department_id, position_id, atasan_id, status)
			VALUES ($1, $2, 'L', '1990-01-01', '2026-01-05', $3, $4, $5, 'aktif')
			RETURNING id
		`, nip, name, departmentID, positionID, supervisor).Scan(&id))
		return id
	}
	insertUser := func(employeeID uuid.UUID, email, role string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `
			INSERT INTO users (employee_id, email, password_hash, role_id)
			VALUES ($1, $2, 'x', (SELECT id FROM roles WHERE nama = $3))
			RETURNING id
		`, employeeID, email, role).Scan(&id))
		return id
	}

	f.supervisorEmployee = insertEmployee("SPV-"+suffix, "Atasan Sintetis", nil)
	f.subordinateEmployee = insertEmployee("SUB-"+suffix, "Karyawan Sintetis", &f.supervisorEmployee)
	f.hrEmployee = insertEmployee("HR-"+suffix, "HR Sintetis", nil)
	f.tmEmployee = insertEmployee("TM-"+suffix, "Top Management Sintetis", nil)

	f.supervisorUser = insertUser(f.supervisorEmployee, "atasan-koreksi-"+suffix+"@example.test", "atasan")
	f.subordinateUser = insertUser(f.subordinateEmployee, "karyawan-koreksi-"+suffix+"@example.test", "karyawan")
	f.hrUser = insertUser(f.hrEmployee, "hr-koreksi-"+suffix+"@example.test", "hr")
	f.tmUser = insertUser(f.tmEmployee, "tm-koreksi-"+suffix+"@example.test", "top_management")

	employeeIDs := []uuid.UUID{f.supervisorEmployee, f.subordinateEmployee, f.hrEmployee, f.tmEmployee}
	userIDs := []uuid.UUID{f.supervisorUser, f.subordinateUser, f.hrUser, f.tmUser}

	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM attendance_correction_approvals WHERE approver_id = ANY($1)`, userIDs)
		_, _ = pool.Exec(c, `DELETE FROM attendance_corrections WHERE user_id = ANY($1)`, userIDs)
		_, _ = pool.Exec(c, `DELETE FROM attendances WHERE user_id = ANY($1)`, userIDs)
		_, _ = pool.Exec(c, `ALTER TABLE audit_logs DISABLE TRIGGER trg_audit_logs_append_only`)
		_, _ = pool.Exec(c, `DELETE FROM audit_logs WHERE user_id = ANY($1)`, userIDs)
		_, _ = pool.Exec(c, `ALTER TABLE audit_logs ENABLE TRIGGER trg_audit_logs_append_only`)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = ANY($1)`, userIDs)
		_, _ = pool.Exec(c, `UPDATE employees SET atasan_id = NULL WHERE id = ANY($1)`, employeeIDs)
		_, _ = pool.Exec(c, `DELETE FROM employees WHERE id = ANY($1)`, employeeIDs)
		_, _ = pool.Exec(c, `DELETE FROM positions WHERE id = $1`, positionID)
		_, _ = pool.Exec(c, `DELETE FROM departments WHERE id = $1`, departmentID)
		pool.Close()
	})
	return f
}

func (f *correctionFixture) call(
	t *testing.T, fn http.HandlerFunc, method, target, body string, identity domain.Identity, vars map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(middleware.WithIdentity(request.Context(), identity))
	if vars != nil {
		request = mux.SetURLVars(request, vars)
	}
	recorder := httptest.NewRecorder()
	fn(recorder, request)
	return recorder
}

func (f *correctionFixture) identity(userID, employeeID uuid.UUID, role domain.RoleName) domain.Identity {
	return domain.Identity{UserID: userID, EmployeeID: employeeID, Role: role}
}

// insertCheckIn menyiapkan absensi asal agar approval final koreksi tidak gagal dengan
// ATTENDANCE_NOT_FOUND (DecideCorrection meng-update baris check_in yang sudah ada).
func (f *correctionFixture) insertCheckIn(t *testing.T, userID uuid.UUID, date string) {
	t.Helper()
	local := date + " 08:00:00"
	_, err := f.pool.Exec(context.Background(), `
		INSERT INTO attendances (user_id, tanggal, tipe, mode_kerja, waktu_network, waktu_local, gps_lat, gps_long, status)
		VALUES ($1, $2::date, 'check_in', 'WFH', $3::timestamp, $3::timestamp, -6.2, 106.8, 'tepat_waktu')
	`, userID, date, local)
	require.NoError(t, err)
}

func decodeCorrectionData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	return payload.Data
}

func decodeCorrectionList(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	return payload.Data
}

func decodeCorrectionMeta(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload struct {
		Meta map[string]any `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	return payload.Meta
}

func containsCorrectionID(items []map[string]any, id string) bool {
	for _, item := range items {
		if item["id"] == id {
			return true
		}
	}
	return false
}

// Bawahan mengajukan koreksi -> Atasan menyetujui -> HR menyetujui. Atasan dan HR harus
// tetap melihat pengajuan ini di histori mereka setelah diputuskan, bukan hanya saat masih
// menjadi antrean aktif. Top Management (yang hanya mengawasi pengajuan HR) tidak boleh
// melihatnya.
func TestCorrectionHistoryStaysVisibleAfterDecision(t *testing.T) {
	f := newCorrectionFixture(t)
	date := "2026-08-14"
	f.insertCheckIn(t, f.subordinateUser, date)

	subordinate := f.identity(f.subordinateUser, f.subordinateEmployee, domain.RoleEmployee)
	supervisor := f.identity(f.supervisorUser, f.supervisorEmployee, domain.RoleSupervisor)
	hr := f.identity(f.hrUser, f.hrEmployee, domain.RoleHR)
	topManagement := f.identity(f.tmUser, f.tmEmployee, domain.RoleTopManagement)

	rec := f.call(t, f.handler.CreateCorrection, http.MethodPost, "/absensi/koreksi",
		`{"tanggal":"`+date+`","waktu_check_in":"09:20","alasan":"Perangkat absensi bermasalah"}`,
		subordinate, nil)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	created := decodeCorrectionData(t, rec)
	assert.Equal(t, "menunggu_atasan", created["status"])
	id := created["id"].(string)

	// Atasan menyetujui tahap pertama.
	rec = f.call(t, f.handler.DecideCorrection, http.MethodPut, "/absensi/koreksi/"+id+"/decision",
		`{"keputusan":"setujui"}`, supervisor, map[string]string{"id": id})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "menunggu_hr", decodeCorrectionData(t, rec)["status"])

	// HR melihat pengajuan ini di antrean aktifnya.
	rec = f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi", "", hr, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, containsCorrectionID(decodeCorrectionList(t, rec), id), "HR harus melihat koreksi menunggu_hr")

	// HR menyetujui keputusan final.
	rec = f.call(t, f.handler.DecideCorrection, http.MethodPut, "/absensi/koreksi/"+id+"/decision",
		`{"keputusan":"setujui"}`, hr, map[string]string{"id": id})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "disetujui", decodeCorrectionData(t, rec)["status"])

	// Atasan tetap melihat riwayat koreksi bawahannya yang sudah disetujui, bukan hanya
	// saat masih menunggu_atasan.
	rec = f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi", "", supervisor, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, containsCorrectionID(decodeCorrectionList(t, rec), id), "Atasan harus tetap melihat histori koreksi bawahan yang sudah diputuskan")

	// Top Management hanya mengawasi pengajuan HR; koreksi karyawan ini tidak boleh muncul.
	rec = f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi", "", topManagement, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.False(t, containsCorrectionID(decodeCorrectionList(t, rec), id), "Top Management tidak boleh melihat koreksi milik Karyawan")
}

// Koreksi yang diajukan HR harus langsung menunggu Top Management (bukan menunggu_hr, yang
// akan membuat HR bisa menyetujui pengajuannya sendiri), dan hanya Top Management yang
// boleh memutuskannya.
func TestHRSubmittedCorrectionRequiresTopManagementDecision(t *testing.T) {
	f := newCorrectionFixture(t)
	date := "2026-08-14"
	f.insertCheckIn(t, f.hrUser, date)

	hr := f.identity(f.hrUser, f.hrEmployee, domain.RoleHR)
	topManagement := f.identity(f.tmUser, f.tmEmployee, domain.RoleTopManagement)
	subordinate := f.identity(f.subordinateUser, f.subordinateEmployee, domain.RoleEmployee)

	rec := f.call(t, f.handler.CreateCorrection, http.MethodPost, "/absensi/koreksi",
		`{"tanggal":"`+date+`","waktu_check_in":"09:20","alasan":"Perangkat absensi bermasalah"}`,
		hr, nil)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	created := decodeCorrectionData(t, rec)
	require.Equal(t, "menunggu_top_management", created["status"])
	id := created["id"].(string)

	// HR tidak boleh memutuskan pengajuannya sendiri.
	rec = f.call(t, f.handler.DecideCorrection, http.MethodPut, "/absensi/koreksi/"+id+"/decision",
		`{"keputusan":"setujui"}`, hr, map[string]string{"id": id})
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// Karyawan biasa tidak melihat koreksi milik HR di riwayatnya sendiri.
	rec = f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi", "", subordinate, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.False(t, containsCorrectionID(decodeCorrectionList(t, rec), id))

	// Top Management melihat dan dapat memutuskan.
	rec = f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi", "", topManagement, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, containsCorrectionID(decodeCorrectionList(t, rec), id))

	rec = f.call(t, f.handler.DecideCorrection, http.MethodPut, "/absensi/koreksi/"+id+"/decision",
		`{"keputusan":"setujui"}`, topManagement, map[string]string{"id": id})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "disetujui", decodeCorrectionData(t, rec)["status"])

	var stage string
	require.NoError(t, f.pool.QueryRow(context.Background(),
		`SELECT tahap FROM attendance_correction_approvals WHERE correction_id = $1`, id).Scan(&stage))
	assert.Equal(t, "top_management", stage)

	// HR memiliki visibilitas monitoring atas seluruh koreksi, termasuk miliknya sendiri
	// yang sudah final.
	rec = f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi", "", hr, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, containsCorrectionID(decodeCorrectionList(t, rec), id))
}

// "Antrean dan riwayat koreksi" harus dipaginasi: limit membatasi jumlah baris per halaman
// dan meta.total_page dihitung dari total_data sesungguhnya, bukan dari panjang page saat ini.
func TestListCorrectionsIsPaginated(t *testing.T) {
	f := newCorrectionFixture(t)
	subordinate := f.identity(f.subordinateUser, f.subordinateEmployee, domain.RoleEmployee)
	hr := f.identity(f.hrUser, f.hrEmployee, domain.RoleHR)

	for _, date := range []string{"2026-08-10", "2026-08-11", "2026-08-12"} {
		rec := f.call(t, f.handler.CreateCorrection, http.MethodPost, "/absensi/koreksi",
			`{"tanggal":"`+date+`","waktu_check_in":"09:20","alasan":"Perangkat absensi bermasalah"}`,
			subordinate, nil)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	}

	rec := f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi?page=1&limit=2", "", hr, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Len(t, decodeCorrectionList(t, rec), 2)
	meta := decodeCorrectionMeta(t, rec)
	assert.Equal(t, float64(1), meta["page"])
	assert.Equal(t, float64(2), meta["limit"])
	assert.Equal(t, float64(3), meta["total_data"])
	assert.Equal(t, float64(2), meta["total_page"])

	rec = f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi?page=2&limit=2", "", hr, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Len(t, decodeCorrectionList(t, rec), 1)
	meta = decodeCorrectionMeta(t, rec)
	assert.Equal(t, float64(2), meta["page"])
	assert.Equal(t, float64(3), meta["total_data"])
}

// Halaman "Koreksi Absensi" pribadi (GET /absensi/koreksi/saya) selalu hanya menunjukkan
// pengajuan milik user yang login sendiri, apa pun rolenya — kontras dengan antrean
// persetujuan (GET /absensi/koreksi) yang untuk Atasan/HR sengaja menunjukkan lebih banyak.
func TestListMyCorrectionsShowsOnlyOwnRegardlessOfRole(t *testing.T) {
	f := newCorrectionFixture(t)
	date := "2026-08-14"

	subordinate := f.identity(f.subordinateUser, f.subordinateEmployee, domain.RoleEmployee)
	supervisor := f.identity(f.supervisorUser, f.supervisorEmployee, domain.RoleSupervisor)
	hr := f.identity(f.hrUser, f.hrEmployee, domain.RoleHR)

	submit := func(identity domain.Identity) string {
		rec := f.call(t, f.handler.CreateCorrection, http.MethodPost, "/absensi/koreksi",
			`{"tanggal":"`+date+`","waktu_check_in":"09:20","alasan":"Perangkat absensi bermasalah"}`,
			identity, nil)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		return decodeCorrectionData(t, rec)["id"].(string)
	}
	subordinateID := submit(subordinate)
	supervisorID := submit(supervisor)
	hrID := submit(hr)

	// Atasan melihat bawahannya di antrean (ListCorrections)...
	rec := f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi", "", supervisor, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, containsCorrectionID(decodeCorrectionList(t, rec), subordinateID))

	// ...tapi riwayat pribadinya (ListMyCorrections) hanya berisi miliknya sendiri.
	rec = f.call(t, f.handler.ListMyCorrections, http.MethodGet, "/absensi/koreksi/saya", "", supervisor, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	own := decodeCorrectionList(t, rec)
	assert.True(t, containsCorrectionID(own, supervisorID))
	assert.False(t, containsCorrectionID(own, subordinateID))

	// HR melihat semuanya di antrean (visibilitas monitoring)...
	rec = f.call(t, f.handler.ListCorrections, http.MethodGet, "/absensi/koreksi", "", hr, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	all := decodeCorrectionList(t, rec)
	assert.True(t, containsCorrectionID(all, subordinateID))
	assert.True(t, containsCorrectionID(all, supervisorID))
	assert.True(t, containsCorrectionID(all, hrID))

	// ...tapi riwayat pribadinya hanya berisi miliknya sendiri, bukan visibilitas global.
	rec = f.call(t, f.handler.ListMyCorrections, http.MethodGet, "/absensi/koreksi/saya", "", hr, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	hrOwn := decodeCorrectionList(t, rec)
	assert.True(t, containsCorrectionID(hrOwn, hrID))
	assert.False(t, containsCorrectionID(hrOwn, subordinateID))
	assert.False(t, containsCorrectionID(hrOwn, supervisorID))
}
