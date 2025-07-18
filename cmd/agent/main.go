package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/infrastructure"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/services"
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
			log.Info("shutting down gracefully...")
			return ctx.Err()
		case <-tickerPoll.C:
			if err = collectService.UpdateAllMetrics(); err != nil {
				log.Error("failed to update metrics", zap.Error(err))
			}
		case <-tickerReport.C:
			queryService.SendMetrics(newClient, log)
			if err = collectService.ResetPollCount(); err != nil {
				log.Error("failed to reset poll count", zap.Error(err))
			}
		}
	}
}
