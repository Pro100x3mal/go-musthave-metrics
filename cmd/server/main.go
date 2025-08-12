package main

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories/retry"
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
		return fmt.Errorf("failed to get config: %w", err)
	}

	logger, err := infrastructure.NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	mainLogger := logger.Named("main")
	srvLogger := logger.Named("server")

	mainLogger.Info("starting application")

	var repo repositories.Repository
	var wg sync.WaitGroup

	switch {
	case cfg.DatabaseDSN != "":
		dbLogger := logger.Named("database")
		dbRepo, err := repositories.NewDB(ctx, cfg, dbLogger)
		if err != nil {
			dbLogger.Error("failed to initialize database storage", zap.Error(err))
			return err
		}
		defer dbRepo.Close()
		repo = retry.NewRepoWithRetry(dbRepo, []time.Duration{}, time.Second)
	case cfg.FileStoragePath != "":
		fsLogger := logger.Named("file_storage")
		msRepo := repositories.NewMemStorage()
		repo, err = repositories.NewFileStorage(ctx, cfg, msRepo, &wg, fsLogger)
		if err != nil {
			fsLogger.Error("failed to initialize file storage", zap.Error(err))
			return err
		}
	default:
		msLogger := logger.Named("memory_storage")
		msLogger.Info("initializing in-memory storage")
		repo = repositories.NewMemStorage()
		msLogger.Info("in-memory storage initialized successfully")
	}

	service := services.NewMetricsService(repo)
	handler := handlers.NewMetricsHandler(service, srvLogger)

	if err = handler.StartServer(ctx, cfg); err != nil {
		srvLogger.Error("server failed", zap.Error(err))
	}

	wg.Wait()
	mainLogger.Info("application stopped gracefully")
	return err
}
