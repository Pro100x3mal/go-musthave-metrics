package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/logger"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/services"
	"go.uber.org/zap"
)

func main() {
	mainLogger := zap.NewExample()
	defer mainLogger.Sync()

	if err := run(); err != nil {
		mainLogger.Fatal("application failed:", zap.Error(err))
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := configs.GetConfig()
	if err != nil {
		return err
	}

	if err = logger.Initialize(cfg); err != nil {
		return err
	}
	defer logger.Log.Sync()

	repo := repositories.NewMemStorage()
	fs, err := repositories.NewFileStorage(cfg, repo)
	if err != nil {
		logger.Log.Error("failed to initialize file storage", zap.Error(err))
		return err
	}
	logger.Log.Info("file storage initialized")

	if cfg.IsRestore {
		logger.Log.Info("restoring metrics from file")
		if err = fs.Restore(); err != nil {
			logger.Log.Error("failed to restore metrics", zap.Error(err))
			return err
		}
		logger.Log.Info("metrics restored successfully")
	}

	var wg sync.WaitGroup
	if cfg.StoreInterval > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runAutoSave(ctx, fs, cfg.StoreInterval)
		}()
	}

	metricsService := services.NewMetricsService(fs)
	metricsHandler := handlers.NewMetricsHandler(metricsService)

	logger.Log.Info("starting application")

	if err = handlers.StartServer(ctx, cfg, metricsHandler); err != nil {
		logger.Log.Error("server failed", zap.Error(err))
	}
	logger.Log.Info("server shutdown complete")

	wg.Wait()

	logger.Log.Info("application stopped gracefully")
	return err
}

func runAutoSave(ctx context.Context, fs *repositories.FileStorage, interval time.Duration) {
	logger.Log.Info("starting auto save loop")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Log.Info("running auto save loop")
			if err := fs.SaveToFile(); err != nil {
				logger.Log.Error("failed to save metrics to file", zap.Error(err))
			}
		case <-ctx.Done():
			logger.Log.Info("stopping auto save loop")
			if err := fs.SaveToFile(); err != nil {
				logger.Log.Error("failed to save metrics to file on shutdown", zap.Error(err))
			}
			if err := fs.Close(); err != nil {
				logger.Log.Error("failed to close file storage", zap.Error(err))
			}
			return
		}
	}
}
