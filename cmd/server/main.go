package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
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

	var wg sync.WaitGroup
	if cfg.StoreInterval > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runAutoSave(ctx, log, fs, cfg.StoreInterval)
		}()
	}

	metricsService := services.NewMetricsService(fs)
	metricsHandler := handlers.NewMetricsHandler(metricsService)

	log.Info("starting application")

	if err = handlers.StartServer(ctx, cfg, log, metricsHandler); err != nil {
		log.Error("server failed", zap.Error(err))
	}
	log.Info("server shutdown complete")

	wg.Wait()

	log.Info("application stopped gracefully")
	return err
}

func runAutoSave(ctx context.Context, log *infrastructure.Logger, fs *repositories.FileStorage, interval time.Duration) {
	log.Info("starting auto save loop")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Info("running auto save loop")
			if err := fs.SaveToFile(); err != nil {
				log.Error("failed to save metrics to file", zap.Error(err))
			}
		case <-ctx.Done():
			log.Info("stopping auto save loop")
			if err := fs.SaveToFile(); err != nil {
				log.Error("failed to save metrics to file on shutdown", zap.Error(err))
			}
			if err := fs.Close(); err != nil {
				log.Error("failed to close file storage", zap.Error(err))
			}
			return
		}
	}
}
