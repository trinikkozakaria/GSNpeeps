// Integration test untuk edit + nonaktif Master Jenis Dokumen (D-041). Menjalankan
// UATHandler langsung dengan identity disuntik ke context — sama seperti
// company_feed_integration_test.go — karena UATHandler memakai *pgxpool.Pool mentah.
// Dilewati kecuali TEST_DATABASE_URL menunjuk ke PostgreSQL bermigrasi.
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

type documentTypeFixture struct {
	pool     *pgxpool.Pool
	handler  *handler.UATHandler
	hrUserID uuid.UUID
}

func newDocumentTypeFixture(t *testing.T) *documentTypeFixture {
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
	var departmentID, positionID, employeeID, userID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO departments (nama) VALUES ($1) RETURNING id`, "Uji Doktipe "+suffix,
	).Scan(&departmentID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO positions (nama, department_id) VALUES ($1, $2) RETURNING id`,
		"Uji Staff "+suffix, departmentID,
	).Scan(&positionID))
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO employees (nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join, department_id, position_id, status)
		VALUES ($1, $2, 'P', '1995-01-01', '2026-01-05', $3, $4, 'aktif')
		RETURNING id
	`, "UJI-DOKTIPE-"+suffix, "HR Uji "+suffix, departmentID, positionID).Scan(&employeeID))
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO users (employee_id, email, password_hash, role_id)
		VALUES ($1, $2, 'x', (SELECT id FROM roles WHERE nama = 'hr'))
		RETURNING id
	`, employeeID, "hr-doktipe-"+suffix+"@example.test").Scan(&userID))

	f := &documentTypeFixture{pool: pool, handler: handler.NewUATHandler(pool), hrUserID: userID}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM audit_logs WHERE user_id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM document_types WHERE kode LIKE $1`, "UJI-"+suffix+"%")
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM employees WHERE id = $1`, employeeID)
		_, _ = pool.Exec(c, `DELETE FROM positions WHERE id = $1`, positionID)
		_, _ = pool.Exec(c, `DELETE FROM departments WHERE id = $1`, departmentID)
		pool.Close()
	})
	return f
}

func (f *documentTypeFixture) call(
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

func (f *documentTypeFixture) hr() domain.Identity {
	return domain.Identity{UserID: f.hrUserID, EmployeeID: uuid.New(), Role: domain.RoleHR}
}

func TestDocumentTypeUpdateAndDeactivateLifecycle(t *testing.T) {
	f := newDocumentTypeFixture(t)
	ctx := context.Background()
	suffix := uuid.NewString()[:8]
	kode := "UJI-" + suffix + "-A"

	// CREATE lewat handler sungguhan supaya baris audit CREATE ikut terverifikasi.
	rec := f.call(t, f.handler.CreateDocumentType, http.MethodPost, "/master/jenis-dokumen",
		`{"kode":"`+kode+`","nama":"Jenis `+suffix+`","wajib":true}`, f.hr(), nil)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	id := created.Data.ID
	require.NotEmpty(t, id)

	// UPDATE: rename.
	rec = f.call(t, f.handler.UpdateDocumentType, http.MethodPut, "/master/jenis-dokumen/"+id,
		`{"kode":"`+kode+`","nama":"Jenis `+suffix+` (revisi)","wajib":false}`, f.hr(),
		map[string]string{"id": id})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var name string
	var wajib bool
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT nama, wajib FROM document_types WHERE id = $1`, id).Scan(&name, &wajib))
	assert.Equal(t, "Jenis "+suffix+" (revisi)", name)
	assert.False(t, wajib)

	// UPDATE id tidak dikenal -> 404.
	rec = f.call(t, f.handler.UpdateDocumentType, http.MethodPut, "/master/jenis-dokumen/"+uuid.NewString(),
		`{"kode":"UJI-`+suffix+`-Z","nama":"Tidak ada","wajib":true}`, f.hr(),
		map[string]string{"id": uuid.NewString()})
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// UPDATE dengan kode duplikat -> 409.
	dupKode := "UJI-" + suffix + "-B"
	rec = f.call(t, f.handler.CreateDocumentType, http.MethodPost, "/master/jenis-dokumen",
		`{"kode":"`+dupKode+`","nama":"Jenis `+suffix+` B","wajib":true}`, f.hr(), nil)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	rec = f.call(t, f.handler.UpdateDocumentType, http.MethodPut, "/master/jenis-dokumen/"+id,
		`{"kode":"`+dupKode+`","nama":"Jenis `+suffix+` (revisi)","wajib":false}`, f.hr(),
		map[string]string{"id": id})
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	// DEACTIVATE: bukan hard delete — baris tetap ada dengan is_active=false.
	rec = f.call(t, f.handler.DeleteDocumentType, http.MethodDelete, "/master/jenis-dokumen/"+id,
		"", f.hr(), map[string]string{"id": id})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var active bool
	var stillThere int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT COUNT(*), bool_or(is_active) FROM document_types WHERE id = $1`, id).Scan(&stillThere, &active))
	assert.Equal(t, 1, stillThere, "deactivate tidak boleh menghapus baris")
	assert.False(t, active)

	// DEACTIVATE id tidak dikenal -> 404.
	rec = f.call(t, f.handler.DeleteDocumentType, http.MethodDelete, "/master/jenis-dokumen/"+uuid.NewString(),
		"", f.hr(), map[string]string{"id": uuid.NewString()})
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// Audit: CREATE lalu UPDATE lalu DELETE untuk id ini.
	rows, err := f.pool.Query(ctx, `SELECT aksi FROM audit_logs WHERE data_id = $1 AND modul = 'master_jenis_dokumen' ORDER BY created_at`, id)
	require.NoError(t, err)
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var a string
		require.NoError(t, rows.Scan(&a))
		actions = append(actions, a)
	}
	assert.Equal(t, []string{"CREATE", "UPDATE", "DELETE"}, actions)
}

func TestDocumentTypeMutationsRejectNonHR(t *testing.T) {
	f := newDocumentTypeFixture(t)
	karyawan := domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee}
	id := uuid.NewString()

	create := f.call(t, f.handler.CreateDocumentType, http.MethodPost, "/master/jenis-dokumen",
		`{"kode":"UJI-X","nama":"X","wajib":true}`, karyawan, nil)
	assert.Equal(t, http.StatusForbidden, create.Code)

	update := f.call(t, f.handler.UpdateDocumentType, http.MethodPut, "/master/jenis-dokumen/"+id,
		`{"kode":"UJI-X","nama":"X","wajib":true}`, karyawan, map[string]string{"id": id})
	assert.Equal(t, http.StatusForbidden, update.Code)

	del := f.call(t, f.handler.DeleteDocumentType, http.MethodDelete, "/master/jenis-dokumen/"+id,
		"", karyawan, map[string]string{"id": id})
	assert.Equal(t, http.StatusForbidden, del.Code)
}
