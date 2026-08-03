package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/events"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/postgres"
	redisstore "github.com/gsnpeeps/gsnpeeps/backend/internal/platform/redis"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/webdav"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/worker"
)

// Interval eksekusi job. Auto-escalation berjalan beberapa menit sekali agar SLA 2x24 jam
// tertangani cepat; retensi foto cukup harian.
const (
	escalationInterval = 5 * time.Minute
	retentionInterval  = 24 * time.Hour
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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
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

	storage, err := webdav.New(cfg.Nextcloud)
	if err != nil {
		logger.Error("nextcloud adapter startup failed", "error", err)
		return 1
	}

	transactions := repository.NewTransactionManager(db.Pool())
	audit := repository.NewAuditRepository(db.Pool())
	publisher := events.NewLoggingPublisher(logger)

	leaveEscalation := worker.NewEscalationJob(
		"ketidakhadiran", repository.NewLeaveRepository(db.Pool()),
		transactions, audit, publisher, domain.EventLeaveAutoEscalated, logger,
	)
	overtimeEscalation := worker.NewEscalationJob(
		"lembur", repository.NewOvertimeRepository(db.Pool()),
		transactions, audit, publisher, domain.EventOvertimeAutoEscalated, logger,
	)
	photoRetention := worker.NewPhotoRetentionJob(
		repository.NewAttendanceRepository(db.Pool()), storage, transactions, logger,
	)

	escalationTicker := time.NewTicker(escalationInterval)
	defer escalationTicker.Stop()
	retentionTicker := time.NewTicker(retentionInterval)
	defer retentionTicker.Stop()

	runEscalation := func() {
		if _, err := leaveEscalation.Run(ctx); err != nil {
			logger.Error("leave escalation job failed", "error", err)
		}
		if _, err := overtimeEscalation.Run(ctx); err != nil {
			logger.Error("overtime escalation job failed", "error", err)
		}
	}
	runRetention := func() {
		if _, err := photoRetention.Run(ctx); err != nil {
			logger.Error("photo retention job failed", "error", err)
		}
	}

	logger.Info("worker started",
		"escalation_interval", escalationInterval, "retention_interval", retentionInterval)
	// Eksekusi awal agar backlog tertangani tanpa menunggu tick pertama. Seluruh job
	// idempotent sehingga aman dijalankan berulang.
	runEscalation()
	runRetention()

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped")
			return 0
		case <-escalationTicker.C:
			runEscalation()
		case <-retentionTicker.C:
			runRetention()
		}
	}
}
