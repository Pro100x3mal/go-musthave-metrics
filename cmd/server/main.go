package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/services"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := configs.GetConfig()
	log, err := infrastructure.NewLogger(cfg)
	if err != nil {
		return err
	}
	defer log.Sync()

	repo := repositories.NewMemStorage()
	fs, err := repositories.NewFileStorage(cfg, repo)
	if err != nil {
		log.Error("failed to initialize file storage", zap.Error(err))
		return err
	}
	log.Info("file storage initialized")

	if cfg.IsRestore {
		log.Info("restoring metrics from file")
		if err = fs.Restore(); err != nil {
			log.Error("failed to restore metrics", zap.Error(err))
			return err
		}
		log.Info("metrics restored successfully")
	}

	if cfg.StoreInterval > 0 {
		go runAutoSave(ctx, log, fs, cfg.StoreInterval)
	}

	metricsService := services.NewMetricsService(fs)
	metricsHandler := handlers.NewMetricsHandler(metricsService)

	log.Info("starting application")

	if err = handlers.StartServer(ctx, cfg, log, metricsHandler); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}

	log.Info("application stopped gracefully")
	return nil
}

func runAutoSave(ctx context.Context, log *infrastructure.Logger, fs *repositories.FileStorage, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fs.SaveToFile(); err != nil {
				log.Error("failed to save metrics to file", zap.Error(err))
			}
		case <-ctx.Done():
			if err := fs.SaveToFile(); err != nil {
				log.Error("failed to save metrics to file on shutdown", zap.Error(err))
			}
			return
		}
	}
}
