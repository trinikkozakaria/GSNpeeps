package router

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/dto"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/handler"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// employeeServiceRecorder mencatat operation mana yang dipanggil sehingga urutan pendaftaran
// route dapat diverifikasi.
type employeeServiceRecorder struct{ called string }

func (s *employeeServiceRecorder) ListDepartments(context.Context) ([]domain.Department, error) {
	s.called = "listDepartments"
	return []domain.Department{}, nil
}

func (s *employeeServiceRecorder) ListPositions(
	context.Context,
	*uuid.UUID,
) ([]domain.Position, error) {
	s.called = "listPositions"
	return []domain.Position{}, nil
}

func (s *employeeServiceRecorder) List(
	context.Context,
	domain.Identity,
	domain.EmployeeFilter,
) (domain.EmployeePage, error) {
	s.called = "list"
	return domain.EmployeePage{Page: 1, Limit: 10}, nil
}

func (s *employeeServiceRecorder) Detail(
	context.Context,
	domain.Identity,
	uuid.UUID,
) (domain.EmployeeDetail, error) {
	s.called = "detail"
	return domain.EmployeeDetail{}, nil
}

func (s *employeeServiceRecorder) Create(
	context.Context,
	domain.Identity,
	dto.CreateEmployeeRequest,
	service.RequestMeta,
) (domain.EmployeeMutationResult, error) {
	s.called = "create"
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeServiceRecorder) Update(
	context.Context,
	domain.Identity,
	uuid.UUID,
	dto.UpdateEmployeeRequest,
	service.RequestMeta,
) (domain.EmployeeMutationResult, error) {
	s.called = "update"
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeServiceRecorder) Deactivate(
	context.Context,
	domain.Identity,
	uuid.UUID,
	service.RequestMeta,
) (domain.EmployeeMutationResult, error) {
	s.called = "deactivate"
	return domain.EmployeeMutationResult{}, nil
}

func (s *employeeServiceRecorder) ListDocuments(
	context.Context,
	domain.Identity,
	uuid.UUID,
) ([]domain.EmployeeDocument, error) {
	s.called = "listDocuments"
	return []domain.EmployeeDocument{}, nil
}

func (s *employeeServiceRecorder) UploadDocument(
	context.Context,
	domain.Identity,
	uuid.UUID,
	service.DocumentUpload,
	service.RequestMeta,
) (domain.EmployeeDocument, error) {
	s.called = "uploadDocument"
	return domain.EmployeeDocument{}, nil
}

func (s *employeeServiceRecorder) Export(
	context.Context,
	domain.Identity,
	domain.EmployeeExportQuery,
	service.RequestMeta,
) (domain.ExportFile, error) {
	s.called = "export"
	return domain.ExportFile{
		FileName:    "karyawan.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     []byte("PK"),
	}, nil
}

func (s *employeeServiceRecorder) UpdatePhoto(
	context.Context, domain.Identity, uuid.UUID, service.PhotoUpload, service.RequestMeta,
) (string, error) {
	s.called = "updatePhoto"
	return "GSNpeeps/employee-photos/foto.jpg", nil
}

type profileServiceRecorder struct{ called string }

func (s *profileServiceRecorder) Me(
	context.Context,
	domain.Identity,
) (domain.EmployeeDetail, error) {
	s.called = "me"
	return domain.EmployeeDetail{}, nil
}

func (s *profileServiceRecorder) Metrics(
	context.Context,
	domain.Identity,
) (domain.PersonalMetrics, error) {
	s.called = "metrics"
	return domain.PersonalMetrics{}, nil
}

type dashboardServiceRecorder struct{ called string }

func (s *dashboardServiceRecorder) Metrics(
	context.Context,
	domain.Identity,
	domain.DashboardPeriodType,
	string,
) (domain.DashboardMetrics, error) {
	s.called = "dashboard"
	return domain.DashboardMetrics{}, nil
}

type permissiveValidator struct{}

func (permissiveValidator) Struct(any) map[string]string { return nil }

type routerFixture struct {
	handler   http.Handler
	employees *employeeServiceRecorder
	profiles  *profileServiceRecorder
	dashboard *dashboardServiceRecorder
}

func newRouterFixture() routerFixture {
	employees := &employeeServiceRecorder{}
	profiles := &profileServiceRecorder{}
	dashboard := &dashboardServiceRecorder{}

	// Middleware autentikasi diganti stub yang menyuntikkan identity HR; tujuan test ini
	// adalah pemetaan route, bukan verifikasi token.
	injectIdentity := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			next.ServeHTTP(writer, request.WithContext(middleware.WithIdentity(
				request.Context(),
				domain.Identity{UserID: uuid.New(), EmployeeID: uuid.New(), Role: domain.RoleHR},
			)))
		})
	}
	passthrough := func(next http.Handler) http.Handler { return next }

	return routerFixture{
		handler: New(
			config.HTTP{MaxBodyBytes: 2 << 20, MaxUploadBytes: 6 << 20, AllowedOrigin: "http://localhost:5173"},
			slog.New(slog.DiscardHandler),
			http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusOK)
			}),
			AuthRoutes{
				Handler:            handler.NewAuthHandler(nil, permissiveValidator{}, false),
				Authenticate:       injectIdentity,
				AuthenticatedLimit: passthrough,
			},
			EmployeeRoutes{
				Handler: handler.NewEmployeeHandler(employees, permissiveValidator{}, false),
			},
			ProfileRoutes{Handler: handler.NewProfileHandler(profiles)},
			DashboardRoutes{Handler: handler.NewDashboardHandler(dashboard)},
			AttendanceRoutes{Handler: handler.NewAttendanceHandler(nil, false)},
			LeaveRoutes{Handler: handler.NewLeaveHandler(nil, permissiveValidator{}, false)},
			OvertimeRoutes{Handler: handler.NewOvertimeHandler(nil, permissiveValidator{}, false)},
			NotificationRoutes{Handler: handler.NewNotificationHandler(nil)},
			AccessRoutes{
				Handler: handler.NewAccessHandler(nil, permissiveValidator{}, false),
				// Test ini memverifikasi pemetaan route, bukan matriks permission.
				RequirePermission: func(string, string) func(http.Handler) http.Handler {
					return func(next http.Handler) http.Handler { return next }
				},
			},
		),
		employees: employees,
		profiles:  profiles,
		dashboard: dashboard,
	}
}

// `/karyawan/export` adalah route literal dan tidak boleh tertangkap pola `/karyawan/{id}`.
func TestExportRouteIsNotCapturedByEmployeeIDPattern(t *testing.T) {
	fixture := newRouterFixture()

	recorder := httptest.NewRecorder()
	fixture.handler.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet, "/api/v1/karyawan/export?format=xlsx", nil,
	))

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Equal(t, "export", fixture.employees.called)
}

func TestEmployeeDetailRouteStillMatchesUUID(t *testing.T) {
	fixture := newRouterFixture()

	recorder := httptest.NewRecorder()
	fixture.handler.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet, "/api/v1/karyawan/"+uuid.NewString(), nil,
	))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "detail", fixture.employees.called)
}

func TestEmployeeDataRoutesAreRegistered(t *testing.T) {
	employeeID := uuid.NewString()
	cases := []struct {
		method string
		path   string
		called func(routerFixture) string
		want   string
	}{
		{http.MethodGet, "/api/v1/master/departemen", func(f routerFixture) string { return f.employees.called }, "listDepartments"},
		{http.MethodGet, "/api/v1/master/jabatan", func(f routerFixture) string { return f.employees.called }, "listPositions"},
		{http.MethodGet, "/api/v1/karyawan", func(f routerFixture) string { return f.employees.called }, "list"},
		{http.MethodGet, "/api/v1/karyawan/" + employeeID + "/dokumen", func(f routerFixture) string { return f.employees.called }, "listDocuments"},
		{http.MethodGet, "/api/v1/profil/saya", func(f routerFixture) string { return f.profiles.called }, "me"},
		{http.MethodGet, "/api/v1/profil/saya/metrik", func(f routerFixture) string { return f.profiles.called }, "metrics"},
		{http.MethodGet, "/api/v1/dashboard/metrik", func(f routerFixture) string { return f.dashboard.called }, "dashboard"},
	}

	for _, testCase := range cases {
		t.Run(testCase.path, func(t *testing.T) {
			fixture := newRouterFixture()

			recorder := httptest.NewRecorder()
			fixture.handler.ServeHTTP(recorder, httptest.NewRequest(testCase.method, testCase.path, nil))

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			assert.Equal(t, testCase.want, testCase.called(fixture))
		})
	}
}

// Endpoint di luar kontrak tidak boleh dilayani.
func TestUnknownEndpointsReturnNotFound(t *testing.T) {
	fixture := newRouterFixture()

	for _, path := range []string{
		"/api/v1/karyawan/" + uuid.NewString() + "/gaji",
		"/api/v1/profil/saya/dokumen",
		"/api/v1/dashboard/hiring-progress",
	} {
		recorder := httptest.NewRecorder()
		fixture.handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

		assert.Equalf(t, http.StatusNotFound, recorder.Code, "path %s harus 404", path)
	}
}
