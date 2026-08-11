package router

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/handler"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
)

type AuthRoutes struct {
	Handler            *handler.AuthHandler
	Authenticate       func(http.Handler) http.Handler
	AuthenticatedLimit func(http.Handler) http.Handler
}

type EmployeeRoutes struct {
	Handler *handler.EmployeeHandler
}

type ProfileRoutes struct {
	Handler *handler.ProfileHandler
}

type DashboardRoutes struct {
	Handler *handler.DashboardHandler
}

type AttendanceRoutes struct {
	Handler *handler.AttendanceHandler
}

type LeaveRoutes struct {
	Handler *handler.LeaveHandler
}

type OvertimeRoutes struct {
	Handler *handler.OvertimeHandler
}

type NotificationRoutes struct {
	Handler *handler.NotificationHandler
}

// AccessRoutes memasang modul AKSES. RequirePermission adalah gerbang kedua berbasis matriks
// permission; service tetap memeriksa role sehingga satu lapisan yang salah konfigurasi tidak
// membuka modul ini.
type AccessRoutes struct {
	Handler           *handler.AccessHandler
	RequirePermission func(module, action string) func(http.Handler) http.Handler
}

func New(
	cfg config.HTTP,
	logger *slog.Logger,
	healthHandler http.Handler,
	auth AuthRoutes,
	employees EmployeeRoutes,
	profiles ProfileRoutes,
	dashboards DashboardRoutes,
	attendances AttendanceRoutes,
	leaves LeaveRoutes,
	overtimes OvertimeRoutes,
	notifications NotificationRoutes,
	access AccessRoutes,
) http.Handler {
	router := mux.NewRouter()
	router.Handle("/health", healthHandler).Methods(http.MethodGet)
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/auth/login", auth.Handler.Login).Methods(http.MethodPost)
	api.HandleFunc("/auth/reset-password", auth.Handler.ResetPassword).Methods(http.MethodPost)

	protected := func(handlerFunc http.HandlerFunc) http.Handler {
		return auth.Authenticate(auth.AuthenticatedLimit(handlerFunc))
	}
	api.Handle("/auth/logout", protected(auth.Handler.Logout)).Methods(http.MethodPost)
	api.Handle("/auth/me", protected(auth.Handler.Me)).Methods(http.MethodGet)
	api.Handle("/auth/me/password", protected(auth.Handler.ChangePassword)).Methods(http.MethodPatch)
	api.Handle("/master/departemen", protected(employees.Handler.ListDepartments)).Methods(http.MethodGet)
	api.Handle("/master/jabatan", protected(employees.Handler.ListPositions)).Methods(http.MethodGet)
	api.Handle("/master/lokasi-kantor", protected(attendances.Handler.ListOfficeLocations)).
		Methods(http.MethodGet)
	api.Handle("/karyawan", protected(employees.Handler.List)).Methods(http.MethodGet)
	api.Handle("/karyawan", protected(employees.Handler.Create)).Methods(http.MethodPost)
	// Route literal harus terdaftar sebelum pola `{id}` agar `/karyawan/export` tidak
	// tertangkap sebagai ID karyawan.
	api.Handle("/karyawan/export", protected(employees.Handler.Export)).Methods(http.MethodGet)
	api.Handle("/karyawan/{id}", protected(employees.Handler.Detail)).Methods(http.MethodGet)
	api.Handle("/karyawan/{id}", protected(employees.Handler.Update)).Methods(http.MethodPut)
	api.Handle("/karyawan/{id}", protected(employees.Handler.Deactivate)).Methods(http.MethodDelete)
	api.Handle("/karyawan/{id}/dokumen", protected(employees.Handler.ListDocuments)).
		Methods(http.MethodGet)
	api.Handle("/karyawan/{id}/dokumen", protected(employees.Handler.UploadDocument)).
		Methods(http.MethodPost)
	api.Handle("/profil/saya", protected(profiles.Handler.Me)).Methods(http.MethodGet)
	api.Handle("/profil/saya/metrik", protected(profiles.Handler.Metrics)).Methods(http.MethodGet)
	api.Handle("/dashboard/metrik", protected(dashboards.Handler.Metrics)).Methods(http.MethodGet)

	api.Handle("/absensi/checkin", protected(attendances.Handler.Record)).Methods(http.MethodPost)
	api.Handle("/absensi/livefeed", protected(attendances.Handler.LiveFeed)).Methods(http.MethodGet)
	// Route literal export didaftarkan sebelum pola lain pada prefix yang sama.
	api.Handle("/laporan/kehadiran/export", protected(attendances.Handler.ExportReport)).
		Methods(http.MethodGet)
	api.Handle("/laporan/kehadiran", protected(attendances.Handler.Report)).Methods(http.MethodGet)

	api.Handle("/master/jenis-izin", protected(leaves.Handler.ListTypes)).Methods(http.MethodGet)
	api.Handle("/master/jenis-izin", protected(leaves.Handler.CreateType)).Methods(http.MethodPost)
	api.Handle("/master/jenis-izin/{id}", protected(leaves.Handler.UpdateType)).Methods(http.MethodPut)

	api.Handle("/ketidakhadiran", protected(leaves.Handler.Create)).Methods(http.MethodPost)
	api.Handle("/ketidakhadiran", protected(leaves.Handler.ListForApproval)).Methods(http.MethodGet)
	api.Handle("/ketidakhadiran/saya", protected(leaves.Handler.ListMine)).Methods(http.MethodGet)
	api.Handle("/ketidakhadiran/{id}", protected(leaves.Handler.Detail)).Methods(http.MethodGet)
	api.Handle("/ketidakhadiran/{id}/decision", protected(leaves.Handler.Decide)).
		Methods(http.MethodPut)
	api.Handle("/ketidakhadiran/{id}/delegate", protected(leaves.Handler.Delegate)).
		Methods(http.MethodPut)

	api.Handle("/lembur", protected(overtimes.Handler.Create)).Methods(http.MethodPost)
	api.Handle("/lembur", protected(overtimes.Handler.List)).Methods(http.MethodGet)
	api.Handle("/lembur/saya", protected(overtimes.Handler.ListMine)).Methods(http.MethodGet)
	api.Handle("/lembur/rekap", protected(overtimes.Handler.Recap)).Methods(http.MethodGet)
	api.Handle("/lembur/{id}", protected(overtimes.Handler.Detail)).Methods(http.MethodGet)
	api.Handle("/lembur/{id}/decision", protected(overtimes.Handler.Decide)).Methods(http.MethodPut)

	// Route literal unread-count didaftarkan sebelum pola `{id}` pada prefix yang sama.
	api.Handle("/notifikasi/unread-count", protected(notifications.Handler.UnreadCount)).
		Methods(http.MethodGet)
	api.Handle("/notifikasi", protected(notifications.Handler.List)).Methods(http.MethodGet)
	api.Handle("/notifikasi/{id}/read", protected(notifications.Handler.MarkRead)).
		Methods(http.MethodPut)
	api.Handle("/notifikasi/{id}", protected(notifications.Handler.Dismiss)).
		Methods(http.MethodDelete)

	guarded := func(module, action string, handlerFunc http.HandlerFunc) http.Handler {
		return auth.Authenticate(auth.AuthenticatedLimit(
			access.RequirePermission(module, action)(handlerFunc),
		))
	}
	api.Handle("/akses/role", guarded("akses", "read", access.Handler.ListRoles)).
		Methods(http.MethodGet)
	api.Handle("/akses/permission", guarded("akses", "read", access.Handler.PermissionMatrix)).
		Methods(http.MethodGet)
	api.Handle("/akses/permission", guarded("akses", "update", access.Handler.UpdatePermission)).
		Methods(http.MethodPut)
	api.Handle("/akses/audit-log", guarded("audit", "read", access.Handler.ListAuditLogs)).
		Methods(http.MethodGet)

	api.PathPrefix("").Handler(http.NotFoundHandler())

	var handler http.Handler = router
	handler = middleware.BodyLimit(cfg.MaxBodyBytes, cfg.MaxUploadBytes)(handler)
	handler = middleware.CORS(cfg.AllowedOrigin)(handler)
	handler = middleware.AccessLog(logger)(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.Recovery(logger)(handler)
	return handler
}
