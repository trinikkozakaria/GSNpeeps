// Integration test untuk pagination, edit, dan hapus Company Feed. Menjalankan handler
// langsung (bukan lewat router penuh) dengan identity disuntik ke context, mengikuti pola
// notification_handler_test.go, karena UATHandler memakai *pgxpool.Pool mentah tanpa lapisan
// repository yang dapat di-mock.
package tests

import (
	"context"
	"encoding/json"
	"fmt"
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

type companyFeedFixture struct {
	pool         *pgxpool.Pool
	handler      *handler.UATHandler
	hrUserID     uuid.UUID
	hrEmployeeID uuid.UUID
}

func newCompanyFeedFixture(t *testing.T) *companyFeedFixture {
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
		`INSERT INTO departments (nama) VALUES ($1) RETURNING id`, "Uji Feed "+suffix,
	).Scan(&departmentID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO positions (nama, department_id) VALUES ($1, $2) RETURNING id`,
		"Uji Staff "+suffix, departmentID,
	).Scan(&positionID))
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO employees (nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join, department_id, position_id, status)
		VALUES ($1, $2, 'P', '1995-01-01', '2026-01-05', $3, $4, 'aktif')
		RETURNING id
	`, "UJI-FEED-"+suffix, "HR Uji "+suffix, departmentID, positionID).Scan(&employeeID))
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO users (employee_id, email, password_hash, role_id)
		VALUES ($1, $2, 'x', (SELECT id FROM roles WHERE nama = 'hr'))
		RETURNING id
	`, employeeID, "hr-feed-"+suffix+"@example.test").Scan(&userID))

	f := &companyFeedFixture{
		pool: pool, handler: handler.NewUATHandler(pool), hrUserID: userID, hrEmployeeID: employeeID,
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM audit_logs WHERE user_id = $1`, userID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM company_feeds WHERE author_id = $1`, userID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM employees WHERE id = $1`, employeeID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM positions WHERE id = $1`, positionID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM departments WHERE id = $1`, departmentID)
		pool.Close()
	})
	return f
}

func (f *companyFeedFixture) request(
	t *testing.T, method, target string, body any, vars map[string]string,
) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	var reader *strings.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		reader = strings.NewReader(string(encoded))
	} else {
		reader = strings.NewReader("")
	}
	request := httptest.NewRequest(method, target, reader)
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(middleware.WithIdentity(request.Context(), domain.Identity{
		UserID: f.hrUserID, EmployeeID: f.hrEmployeeID, Role: domain.RoleHR,
	}))
	if vars != nil {
		request = mux.SetURLVars(request, vars)
	}
	recorder := httptest.NewRecorder()
	return recorder, request
}

func decodeEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	return envelope
}

// TestCompanyFeedPaginationCreateUpdateDelete menutupi seluruh siklus yang dipakai halaman
// Beranda (infinite scroll, 20/halaman) dan Company Feed (pagination + edit + hapus).
func TestCompanyFeedPaginationCreateUpdateDelete(t *testing.T) {
	f := newCompanyFeedFixture(t)
	ctx := context.Background()

	// Feed pertama dibuat lewat handler CreateFeed sungguhan (bukan INSERT langsung) supaya
	// baris audit CREATE-nya juga ikut terverifikasi nanti. 22 feed tambahan lewat SQL
	// langsung sekadar padding agar halaman pertama (limit default 20) tidak menghabiskan
	// seluruh dataset, membuktikan total_data/total_page dihitung dari semua baris.
	recorder, request := f.request(t, http.MethodPost, "/company-feed",
		map[string]string{"judul": "Feed Uji", "konten_html": "<p>Konten</p>"}, nil,
	)
	f.handler.CreateFeed(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	created := decodeEnvelope(t, recorder)
	createdData, _ := created["data"].(map[string]any)
	target, err := uuid.Parse(fmt.Sprint(createdData["id"]))
	require.NoError(t, err)

	for i := 0; i < 22; i++ {
		_, err := f.pool.Exec(ctx, `
			INSERT INTO company_feeds (author_id, judul, konten_html) VALUES ($1, $2, $3)
		`, f.hrUserID, "Feed Uji", "<p>Konten</p>")
		require.NoError(t, err)
	}

	recorder, request = f.request(t, http.MethodGet, "/company-feed", nil, nil)
	f.handler.ListFeeds(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	envelope := decodeEnvelope(t, recorder)
	items, _ := envelope["data"].([]any)
	assert.Len(t, items, 20, "default limit halaman pertama harus 20 (kebutuhan infinite scroll Beranda)")
	meta, _ := envelope["meta"].(map[string]any)
	require.NotNil(t, meta, "list harus mengembalikan meta pagination")
	assert.EqualValues(t, 23, meta["total_data"])
	assert.EqualValues(t, 2, meta["total_page"])
	assert.EqualValues(t, 1, meta["page"])
	assert.EqualValues(t, 20, meta["limit"])

	recorder, request = f.request(t, http.MethodGet, "/company-feed?page=2&limit=20", nil, nil)
	f.handler.ListFeeds(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	envelope = decodeEnvelope(t, recorder)
	items, _ = envelope["data"].([]any)
	assert.Len(t, items, 3, "sisa 3 feed pada halaman kedua")

	// Update: HR mana pun (bukan hanya penulis) boleh menyunting.
	recorder, request = f.request(t, http.MethodPut, "/company-feed/"+target.String(),
		map[string]string{"judul": "Judul Diperbarui", "konten_html": "<p>Baru</p>"},
		map[string]string{"id": target.String()},
	)
	f.handler.UpdateFeed(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var storedTitle string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT judul FROM company_feeds WHERE id = $1`, target).Scan(&storedTitle))
	assert.Equal(t, "Judul Diperbarui", storedTitle)

	// Update pada id yang tidak ada -> 404, bukan diam-diam sukses.
	recorder, request = f.request(t, http.MethodPut, "/company-feed/"+uuid.NewString(),
		map[string]string{"judul": "X", "konten_html": "<p>X</p>"},
		map[string]string{"id": uuid.NewString()},
	)
	f.handler.UpdateFeed(recorder, request)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	// Delete: baris hilang dari database dan tercatat di audit_logs.
	recorder, request = f.request(t, http.MethodDelete, "/company-feed/"+target.String(), nil,
		map[string]string{"id": target.String()},
	)
	f.handler.DeleteFeed(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var remaining int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT COUNT(*) FROM company_feeds WHERE id = $1`, target).Scan(&remaining))
	assert.Equal(t, 0, remaining)

	var auditActions []string
	rows, err := f.pool.Query(ctx, `SELECT aksi FROM audit_logs WHERE data_id = $1 ORDER BY created_at`, target)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var action string
		require.NoError(t, rows.Scan(&action))
		auditActions = append(auditActions, action)
	}
	assert.Equal(t, []string{"CREATE", "UPDATE", "DELETE"}, auditActions)

	// Delete pada id yang sudah tidak ada -> 404.
	recorder, request = f.request(t, http.MethodDelete, "/company-feed/"+target.String(), nil,
		map[string]string{"id": target.String()},
	)
	f.handler.DeleteFeed(recorder, request)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
