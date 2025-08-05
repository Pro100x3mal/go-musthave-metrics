package main

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure"
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

	logger, err := infrastructure.NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	var repo repositories.Repository
	var wg sync.WaitGroup

	switch {
	case cfg.DatabaseDSN != "":
		dbRepo, err := repositories.NewDB(ctx, cfg, logger)
		if err != nil {
			return err
		}
		defer dbRepo.Close()
		repo = dbRepo
	case cfg.FileStoragePath != "":
		msRepo := repositories.NewMemStorage()
		repo, err = repositories.NewFileStorage(ctx, cfg, msRepo, &wg, logger)
		if err != nil {
			return err
		}
	default:
		logger.Info("initializing in-memory storage")
		repo = repositories.NewMemStorage()
		logger.Info("in-memory storage initialized successfully")
	}

	service := services.NewMetricsService(repo)
	handler := handlers.NewMetricsHandler(service, logger)

	logger.Info("starting application")

	if err = handler.StartServer(ctx, cfg); err != nil {
		logger.Error("server failed", zap.Error(err))
	}

	wg.Wait()
	logger.Info("application stopped gracefully")
	return err
}
