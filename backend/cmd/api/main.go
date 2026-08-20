package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/handler"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/validation"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/filestore"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/password"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/postgres"
	redisstore "github.com/gsnpeeps/gsnpeeps/backend/internal/platform/redis"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/token"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/router"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/service"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		return 1
	}

	logger := newLogger(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.Postgres)
	if err != nil {
		logger.Error("postgres startup failed", "error", err)
		return 1
	}
	defer db.Close()

	cache, err := redisstore.Open(ctx, cfg.Redis)
	if err != nil {
		logger.Error("redis startup failed", "error", err)
		return 1
	}
	defer cache.Close()

	documentStore, err := filestore.New(ctx, cfg)
	if err != nil {
		logger.Error("object storage adapter startup failed", "error", err, "driver", cfg.Storage.Driver)
		return 1
	}

	healthService := service.NewHealthService(db, cache)
	healthHandler := handler.NewHealthHandler(healthService)

	authRepository := repository.NewAuthRepository(db.Pool())
	auditRepository := repository.NewAuditRepository(db.Pool())
	passwordHasher := password.New(cfg.Auth)
	tokenManager := token.New(cfg.JWT)
	sessionStore := redisstore.NewSessionStore(cache.Raw())
	rateLimiter := redisstore.NewRateLimiter(cache.Raw())
	authService, err := service.NewAuthService(
		authRepository,
		passwordHasher,
		tokenManager,
		sessionStore,
		rateLimiter,
		auditRepository,
		cfg.Auth,
	)
	if err != nil {
		logger.Error("auth service startup failed", "error", err)
		return 1
	}
	authHandler := handler.NewAuthHandler(authService, validation.New(), cfg.HTTP.TrustProxy)
	employeeRepository := repository.NewEmployeeRepository(db.Pool())
	transactionManager := repository.NewTransactionManager(db.Pool())
	employeeService := service.NewEmployeeService(
		employeeRepository,
		transactionManager,
		auditRepository,
		sessionStore,
		passwordHasher,
		documentStore,
	)
	employeeHandler := handler.NewEmployeeHandler(employeeService, validation.New(), cfg.HTTP.TrustProxy)

	// Metrik kini dibaca dari tabel absensi, ketidakhadiran, dan lembur yang sudah tersedia,
	// menggantikan adapter sementara D-020.
	attendanceRepository := repository.NewAttendanceRepository(db.Pool())
	leaveRepository := repository.NewLeaveRepository(db.Pool())
	overtimeRepository := repository.NewOvertimeRepository(db.Pool())

	profileHandler := handler.NewProfileHandler(
		service.NewProfileService(employeeRepository, attendanceRepository),
	)
	dashboardHandler := handler.NewDashboardHandler(service.NewDashboardService(
		repository.NewDashboardRepository(db.Pool()),
		attendanceRepository,
		overtimeRepository,
	))

	// Notifikasi ditulis di dalam transaction keputusan approval (D-033).
	notificationRepository := repository.NewNotificationRepository(db.Pool())
	eventPublisher := service.NewNotificationPublisher(notificationRepository, logger)
	notificationHandler := handler.NewNotificationHandler(
		service.NewNotificationService(notificationRepository),
	)

	permissionRepository := repository.NewPermissionRepository(db.Pool())
	permissionCache := redisstore.NewPermissionCache(cache.Raw())
	permissionChecker := service.NewCachedPermissionChecker(
		permissionRepository, permissionCache, logger,
	)
	accessHandler := handler.NewAccessHandler(
		service.NewAccessService(
			permissionRepository, auditRepository, transactionManager,
			auditRepository, permissionCache,
		),
		validation.New(), cfg.HTTP.TrustProxy,
	)

	attendanceHandler := handler.NewAttendanceHandler(
		service.NewAttendanceService(
			attendanceRepository, transactionManager, auditRepository, documentStore,
		),
		cfg.HTTP.TrustProxy,
	)
	leaveHandler := handler.NewLeaveHandler(
		service.NewLeaveService(
			leaveRepository, employeeRepository, transactionManager, auditRepository,
			documentStore, eventPublisher,
		),
		validation.New(), cfg.HTTP.TrustProxy,
	)
	overtimeHandler := handler.NewOvertimeHandler(
		service.NewOvertimeService(
			overtimeRepository, employeeRepository, transactionManager, auditRepository,
			documentStore, eventPublisher,
		),
		validation.New(), cfg.HTTP.TrustProxy,
	)

	httpRouter := router.New(cfg.HTTP, logger, healthHandler, router.AuthRoutes{
		Handler:            authHandler,
		Authenticate:       middleware.Authenticate(tokenManager, sessionStore),
		AuthenticatedLimit: middleware.AuthenticatedRateLimit(rateLimiter, cfg.Auth.RequestLimit, cfg.Auth.RequestWindow),
	}, router.EmployeeRoutes{
		Handler: employeeHandler,
	}, router.ProfileRoutes{
		Handler: profileHandler,
	}, router.DashboardRoutes{
		Handler: dashboardHandler,
	}, router.AttendanceRoutes{
		Handler: attendanceHandler,
	}, router.LeaveRoutes{
		Handler: leaveHandler,
	}, router.OvertimeRoutes{
		Handler: overtimeHandler,
	}, router.NotificationRoutes{
		Handler: notificationHandler,
	}, router.AccessRoutes{
		Handler: accessHandler,
		RequirePermission: func(module, action string) func(http.Handler) http.Handler {
			return middleware.RequirePermission(permissionChecker, module, action)
		},
	}, router.WithUAT(handler.NewUATHandler(db.Pool(), documentStore)))

	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           httpRouter,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api started", "address", cfg.HTTP.Address, "environment", cfg.App.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api stopped unexpectedly", "error", err)
			return 1
		}
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return 1
	}

	logger.Info("api stopped")
	return 0
}

func newLogger(cfg config.Config) *slog.Logger {
	level := slog.LevelInfo
	if cfg.App.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	options := &slog.HandlerOptions{Level: level}
	if cfg.App.Environment == "development" {
		return slog.New(slog.NewTextHandler(os.Stdout, options))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, options))
}
