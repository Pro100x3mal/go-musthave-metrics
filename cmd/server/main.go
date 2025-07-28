package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

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

	var wg sync.WaitGroup

	ms := repositories.NewMemStorage()
	repo, err := repositories.NewFileStorage(ctx, cfg, ms, &wg)
	if err != nil {
		return err
	}

	metricsService := services.NewMetricsService(repo)
	metricsHandler := handlers.NewMetricsHandler(metricsService)

	logger.Log.Info("starting application")

	if err = handlers.StartServer(ctx, cfg, metricsHandler); err != nil {
		logger.Log.Error("server failed", zap.Error(err))
	}

	wg.Wait()
	logger.Log.Info("application stopped gracefully")
	return err
}
