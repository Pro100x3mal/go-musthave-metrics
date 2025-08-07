package main

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/infrastructure"
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

	logger, err := infrastructure.NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	repo := repositories.NewMemStorage()
	collectService := services.NewMetricsCollectService(repo)
	queryService := services.NewMetricsQueryService(repo, logger)

	newClient := services.NewClient(cfg)

	tickerPoll := time.NewTicker(cfg.PollInterval)
	tickerReport := time.NewTicker(cfg.ReportInterval)
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received, waiting for operations to complete...")
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				logger.Info("all operations completed, shutting down gracefully")
			case <-time.After(10 * time.Second):
				logger.Warn("shutdown timed out, forcefully shutting down...")
			}

			return ctx.Err()

		case <-tickerPoll.C:
			if ctx.Err() != nil {
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err = collectService.UpdateAllMetrics(ctx); err != nil {
					logger.Error("failed to update metrics", zap.Error(err))
				}
			}()

		case <-tickerReport.C:
			if ctx.Err() != nil {
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				queryService.SendMetrics(ctx, newClient)
				if err = collectService.ResetPollCount(); err != nil {
					logger.Error("failed to reset poll count", zap.Error(err))
				}
			}()
		}
	}
}
