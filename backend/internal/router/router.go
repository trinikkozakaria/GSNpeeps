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

func New(
	cfg config.HTTP,
	logger *slog.Logger,
	healthHandler http.Handler,
	auth AuthRoutes,
	employees EmployeeRoutes,
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
	api.Handle("/karyawan", protected(employees.Handler.List)).Methods(http.MethodGet)
	api.Handle("/karyawan", protected(employees.Handler.Create)).Methods(http.MethodPost)
	api.Handle("/karyawan/{id}", protected(employees.Handler.Detail)).Methods(http.MethodGet)
	api.Handle("/karyawan/{id}", protected(employees.Handler.Update)).Methods(http.MethodPut)
	api.Handle("/karyawan/{id}", protected(employees.Handler.Deactivate)).Methods(http.MethodDelete)
	api.PathPrefix("").Handler(http.NotFoundHandler())

	var handler http.Handler = router
	handler = middleware.BodyLimit(cfg.MaxBodyBytes)(handler)
	handler = middleware.CORS(cfg.AllowedOrigin)(handler)
	handler = middleware.AccessLog(logger)(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.Recovery(logger)(handler)
	return handler
}
