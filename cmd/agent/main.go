package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/logger"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/services"
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
	collectService := services.NewMetricsCollectService(repo)
	queryService := services.NewMetricsQueryService(repo)

	newClient := services.NewClient(cfg)

	tickerPoll := time.NewTicker(cfg.PollInterval)
	tickerReport := time.NewTicker(cfg.ReportInterval)
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("shutting down gracefully...")
			return ctx.Err()
		case <-tickerPoll.C:
			if err = collectService.UpdateAllMetrics(); err != nil {
				logger.Log.Error("failed to update metrics", zap.Error(err))
			}
		case <-tickerReport.C:
			queryService.SendMetrics(newClient)
			if err = collectService.ResetPollCount(); err != nil {
				logger.Log.Error("failed to reset poll count", zap.Error(err))
			}
		}
	}
}
