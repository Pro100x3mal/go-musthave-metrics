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

	var wg sync.WaitGroup

	ms := repositories.NewMemStorage()
	repo, err := repositories.NewFileStorage(ctx, cfg, ms, &wg, logger)
	if err != nil {
		return err
	}

	metricsService := services.NewMetricsService(repo)
	metricsHandler := handlers.NewMetricsHandler(metricsService, logger)

	logger.Info("starting application")

	if err = handlers.StartServer(ctx, cfg, metricsHandler); err != nil {
		logger.Error("server failed", zap.Error(err))
	}

	wg.Wait()
	logger.Info("application stopped gracefully")
	return err
}
