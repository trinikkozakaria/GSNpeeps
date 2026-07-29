package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/postgres"
	redisstore "github.com/gsnpeeps/gsnpeeps/backend/internal/platform/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.Postgres)
	if err != nil {
		logger.Error("postgres startup failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	cache, err := redisstore.Open(ctx, cfg.Redis)
	if err != nil {
		logger.Error("redis startup failed", "error", err)
		os.Exit(1)
	}
	defer cache.Close()

	logger.Info("worker ready", "message", "scheduled jobs are intentionally not enabled in the foundation phase")
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	_ = shutdownCtx
	logger.Info("worker stopped")
}
