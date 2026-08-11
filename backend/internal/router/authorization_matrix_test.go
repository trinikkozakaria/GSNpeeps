package router

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/handler"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// operation adalah satu operasi kontrak beserta sifat aksesnya.
type operation struct {
	method string
	path   string
	public bool
}

// contractOperations mendaftar seluruh 47 operasi OpenAPI. Daftar ini menjadi matriks
// otorisasi: setiap operasi terproteksi wajib menolak permintaan tanpa token, dan setiap
// operasi publik wajib tidak menolaknya. Operasi baru yang lupa didaftarkan pada router akan
// menghasilkan 404 dan membuat matriks gagal.
func contractOperations() []operation {
	employeeID := uuid.NewString()
	requestID := uuid.NewString()
	notificationID := uuid.NewString()

	return []operation{
		{http.MethodGet, "/health", true},
		{http.MethodPost, "/api/v1/auth/login", true},
		{http.MethodPost, "/api/v1/auth/reset-password", true},

		{http.MethodPost, "/api/v1/auth/logout", false},
		{http.MethodGet, "/api/v1/auth/me", false},
		{http.MethodPatch, "/api/v1/auth/me/password", false},

		{http.MethodGet, "/api/v1/master/departemen", false},
		{http.MethodGet, "/api/v1/master/jabatan", false},
		{http.MethodGet, "/api/v1/master/lokasi-kantor", false},

		{http.MethodGet, "/api/v1/karyawan", false},
		{http.MethodPost, "/api/v1/karyawan", false},
		{http.MethodGet, "/api/v1/karyawan/export", false},
		{http.MethodGet, "/api/v1/karyawan/" + employeeID, false},
		{http.MethodPut, "/api/v1/karyawan/" + employeeID, false},
		{http.MethodDelete, "/api/v1/karyawan/" + employeeID, false},
		{http.MethodGet, "/api/v1/karyawan/" + employeeID + "/dokumen", false},
		{http.MethodPost, "/api/v1/karyawan/" + employeeID + "/dokumen", false},

		{http.MethodGet, "/api/v1/profil/saya", false},
		{http.MethodGet, "/api/v1/profil/saya/metrik", false},
		{http.MethodGet, "/api/v1/dashboard/metrik", false},

		{http.MethodPost, "/api/v1/absensi/checkin", false},
		{http.MethodGet, "/api/v1/absensi/livefeed", false},
		{http.MethodGet, "/api/v1/laporan/kehadiran", false},
		{http.MethodGet, "/api/v1/laporan/kehadiran/export", false},

		{http.MethodPost, "/api/v1/ketidakhadiran", false},
		{http.MethodGet, "/api/v1/ketidakhadiran", false},
		{http.MethodGet, "/api/v1/ketidakhadiran/saya", false},
		{http.MethodGet, "/api/v1/ketidakhadiran/" + requestID, false},
		{http.MethodPut, "/api/v1/ketidakhadiran/" + requestID + "/decision", false},
		{http.MethodPut, "/api/v1/ketidakhadiran/" + requestID + "/delegate", false},

		{http.MethodGet, "/api/v1/master/jenis-izin", false},
		{http.MethodPost, "/api/v1/master/jenis-izin", false},
		{http.MethodPut, "/api/v1/master/jenis-izin/" + requestID, false},

		{http.MethodPost, "/api/v1/lembur", false},
		{http.MethodGet, "/api/v1/lembur", false},
		{http.MethodGet, "/api/v1/lembur/saya", false},
		{http.MethodGet, "/api/v1/lembur/rekap", false},
		{http.MethodGet, "/api/v1/lembur/" + requestID, false},
		{http.MethodPut, "/api/v1/lembur/" + requestID + "/decision", false},

		{http.MethodGet, "/api/v1/akses/role", false},
		{http.MethodGet, "/api/v1/akses/permission", false},
		{http.MethodPut, "/api/v1/akses/permission", false},
		{http.MethodGet, "/api/v1/akses/audit-log", false},

		{http.MethodGet, "/api/v1/notifikasi", false},
		{http.MethodGet, "/api/v1/notifikasi/unread-count", false},
		{http.MethodPut, "/api/v1/notifikasi/" + notificationID + "/read", false},
		{http.MethodDelete, "/api/v1/notifikasi/" + notificationID, false},
	}
}

// rejectingTokenVerifier menolak token apa pun. Permintaan tanpa header Authorization sudah
// ditolak lebih dahulu, sehingga stub ini cukup untuk kedua kasus.
type rejectingTokenVerifier struct{}

func (rejectingTokenVerifier) Verify(
	context.Context, string,
) (domain.Identity, string, error) {
	return domain.Identity{}, "", errors.New("token tidak valid")
}

type rejectingSessionValidator struct{}

func (rejectingSessionValidator) Validate(context.Context, uuid.UUID, string) error {
	return domain.ErrSessionInvalid
}

// newGuardedRouter memakai middleware autentikasi sungguhan sehingga hasilnya mencerminkan
// perilaku produksi untuk token yang hilang maupun tidak valid.
func newGuardedRouter() http.Handler {
	passthrough := func(next http.Handler) http.Handler { return next }
	return New(
		config.HTTP{MaxBodyBytes: 2 << 20, MaxUploadBytes: 6 << 20},
		slog.New(slog.DiscardHandler),
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}),
		AuthRoutes{
			Handler:            handler.NewAuthHandler(nil, permissiveValidator{}, false),
			Authenticate:       middleware.Authenticate(rejectingTokenVerifier{}, rejectingSessionValidator{}),
			AuthenticatedLimit: passthrough,
		},
		EmployeeRoutes{Handler: handler.NewEmployeeHandler(nil, permissiveValidator{}, false)},
		ProfileRoutes{Handler: handler.NewProfileHandler(nil)},
		DashboardRoutes{Handler: handler.NewDashboardHandler(nil)},
		AttendanceRoutes{Handler: handler.NewAttendanceHandler(nil, false)},
		LeaveRoutes{Handler: handler.NewLeaveHandler(nil, permissiveValidator{}, false)},
		OvertimeRoutes{Handler: handler.NewOvertimeHandler(nil, permissiveValidator{}, false)},
		NotificationRoutes{Handler: handler.NewNotificationHandler(nil)},
		AccessRoutes{
			Handler:           handler.NewAccessHandler(nil, permissiveValidator{}, false),
			RequirePermission: func(string, string) func(http.Handler) http.Handler { return passthrough },
		},
	)
}

// Jumlah operasi harus tetap 47 sesuai keputusan D-001 dan D-035; endpoint baru memerlukan
// revisi kontrak lebih dahulu.
func TestContractExposesExactlyFortySevenOperations(t *testing.T) {
	assert.Len(t, contractOperations(), 47)
}

// Setiap operasi terproteksi menolak permintaan tanpa token. Karena `/api/v1` memiliki
// NotFoundHandler, 401 sekaligus membuktikan route terdaftar; route yang lupa didaftarkan
// akan menghasilkan 404.
func TestEveryProtectedOperationRejectsMissingToken(t *testing.T) {
	router := newGuardedRouter()

	for _, candidate := range contractOperations() {
		if candidate.public {
			continue
		}
		t.Run(candidate.method+" "+candidate.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(candidate.method, candidate.path, nil))

			require.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
			assert.Contains(t, recorder.Body.String(), "INVALID_TOKEN")
		})
	}
}

// Kontrol untuk matriks di atas: path yang tidak terdaftar menghasilkan 404, bukan 401.
// Tanpa ini, 401 pada operasi terdaftar tidak membuktikan apa pun.
func TestUnregisteredPathReturnsNotFoundInsteadOfUnauthorized(t *testing.T) {
	router := newGuardedRouter()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet, "/api/v1/master/lokasi-gudang", nil,
	))

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestEveryProtectedOperationRejectsInvalidToken(t *testing.T) {
	router := newGuardedRouter()

	for _, candidate := range contractOperations() {
		if candidate.public {
			continue
		}
		t.Run(candidate.method+" "+candidate.path, func(t *testing.T) {
			request := httptest.NewRequest(candidate.method, candidate.path, nil)
			request.Header.Set("Authorization", "Bearer token-kedaluwarsa")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
		})
	}
}

// Endpoint publik tidak boleh ikut terkunci autentikasi.
func TestPublicOperationsDoNotRequireToken(t *testing.T) {
	router := newGuardedRouter()

	for _, candidate := range contractOperations() {
		if !candidate.public {
			continue
		}
		t.Run(candidate.method+" "+candidate.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(candidate.method, candidate.path, nil))

			assert.NotEqual(t, http.StatusUnauthorized, recorder.Code)
			assert.NotEqual(t, http.StatusNotFound, recorder.Code)
		})
	}
}

// Token yang lolos verifikasi tetapi sesinya sudah dicabut (logout atau lockout) ditolak.
type acceptingTokenVerifier struct{ role domain.RoleName }

func (v acceptingTokenVerifier) Verify(
	context.Context, string,
) (domain.Identity, string, error) {
	return domain.Identity{
		UserID:     uuid.New(),
		EmployeeID: uuid.New(),
		Role:       v.role,
	}, "fingerprint", nil
}

func TestLoggedOutSessionIsRejectedOnEveryProtectedOperation(t *testing.T) {
	passthrough := func(next http.Handler) http.Handler { return next }
	router := New(
		config.HTTP{MaxBodyBytes: 2 << 20, MaxUploadBytes: 6 << 20},
		slog.New(slog.DiscardHandler),
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}),
		AuthRoutes{
			Handler: handler.NewAuthHandler(nil, permissiveValidator{}, false),
			Authenticate: middleware.Authenticate(
				acceptingTokenVerifier{role: domain.RoleHR}, rejectingSessionValidator{},
			),
			AuthenticatedLimit: passthrough,
		},
		EmployeeRoutes{Handler: handler.NewEmployeeHandler(nil, permissiveValidator{}, false)},
		ProfileRoutes{Handler: handler.NewProfileHandler(nil)},
		DashboardRoutes{Handler: handler.NewDashboardHandler(nil)},
		AttendanceRoutes{Handler: handler.NewAttendanceHandler(nil, false)},
		LeaveRoutes{Handler: handler.NewLeaveHandler(nil, permissiveValidator{}, false)},
		OvertimeRoutes{Handler: handler.NewOvertimeHandler(nil, permissiveValidator{}, false)},
		NotificationRoutes{Handler: handler.NewNotificationHandler(nil)},
		AccessRoutes{
			Handler:           handler.NewAccessHandler(nil, permissiveValidator{}, false),
			RequirePermission: func(string, string) func(http.Handler) http.Handler { return passthrough },
		},
	)

	for _, candidate := range contractOperations() {
		if candidate.public {
			continue
		}
		t.Run(candidate.method+" "+candidate.path, func(t *testing.T) {
			request := httptest.NewRequest(candidate.method, candidate.path, nil)
			request.Header.Set("Authorization", "Bearer token-valid")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
		})
	}
}

// denyingPermissionChecker menolak seluruh kapabilitas, meniru Karyawan atau Atasan yang
// mencoba membuka URL modul AKSES secara langsung.
type denyingPermissionChecker struct{ calls []string }

func (c *denyingPermissionChecker) HasPermission(
	_ context.Context, _ domain.RoleName, module, action string,
) (bool, error) {
	c.calls = append(c.calls, module+"."+action)
	return false, nil
}

// Modul AKSES memiliki gerbang matriks permission sebelum handler dijalankan.
func TestAccessRoutesAreGuardedByPermissionMatrix(t *testing.T) {
	checker := &denyingPermissionChecker{}
	passthrough := func(next http.Handler) http.Handler { return next }
	injectIdentity := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			next.ServeHTTP(writer, request.WithContext(middleware.WithIdentity(
				request.Context(),
				domain.Identity{
					UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleEmployee,
				},
			)))
		})
	}

	router := New(
		config.HTTP{MaxBodyBytes: 2 << 20, MaxUploadBytes: 6 << 20},
		slog.New(slog.DiscardHandler),
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}),
		AuthRoutes{
			Handler:            handler.NewAuthHandler(nil, permissiveValidator{}, false),
			Authenticate:       injectIdentity,
			AuthenticatedLimit: passthrough,
		},
		EmployeeRoutes{Handler: handler.NewEmployeeHandler(nil, permissiveValidator{}, false)},
		ProfileRoutes{Handler: handler.NewProfileHandler(nil)},
		DashboardRoutes{Handler: handler.NewDashboardHandler(nil)},
		AttendanceRoutes{Handler: handler.NewAttendanceHandler(nil, false)},
		LeaveRoutes{Handler: handler.NewLeaveHandler(nil, permissiveValidator{}, false)},
		OvertimeRoutes{Handler: handler.NewOvertimeHandler(nil, permissiveValidator{}, false)},
		NotificationRoutes{Handler: handler.NewNotificationHandler(nil)},
		AccessRoutes{
			Handler: handler.NewAccessHandler(nil, permissiveValidator{}, false),
			RequirePermission: func(module, action string) func(http.Handler) http.Handler {
				return middleware.RequirePermission(checker, module, action)
			},
		},
	)

	cases := []struct {
		method     string
		path       string
		capability string
	}{
		{http.MethodGet, "/api/v1/akses/role", "akses.read"},
		{http.MethodGet, "/api/v1/akses/permission", "akses.read"},
		{http.MethodPut, "/api/v1/akses/permission", "akses.update"},
		{http.MethodGet, "/api/v1/akses/audit-log", "audit.read"},
	}

	for _, testCase := range cases {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			checker.calls = nil

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(testCase.method, testCase.path, nil))

			require.Equal(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
			assert.Contains(t, recorder.Body.String(), "FORBIDDEN")
			// Handler tidak pernah dijalankan, sehingga mutation yang ditolak tidak
			// menghasilkan side effect apa pun.
			assert.Equal(t, []string{testCase.capability}, checker.calls)
		})
	}
}

// Route notifikasi literal tidak boleh tertangkap pola `{id}`.
func TestUnreadCountRouteIsNotCapturedByNotificationIDPattern(t *testing.T) {
	router := newGuardedRouter()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet, "/api/v1/notifikasi/unread-count", nil,
	))

	// Tanpa token hasilnya 401; yang penting bukan 404 maupun 405 dari pola yang salah.
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}
