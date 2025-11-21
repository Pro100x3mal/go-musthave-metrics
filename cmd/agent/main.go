package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/infrastructure"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/services"
	"github.com/Pro100x3mal/go-musthave-metrics/pkg/crypto"
	"go.uber.org/zap"
)

func main() {
	mainLogger := zap.NewExample()
	defer mainLogger.Sync()

	mainLogger.Info("starting application",
		zap.String("build version", models.BuildVersion),
		zap.String("build date", models.BuildDate),
		zap.String("build commit", models.BuildCommit),
	)

	if err := run(); err != nil {
		mainLogger.Fatal("application failed:", zap.Error(err))
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
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
	queryService := services.NewMetricsQueryService(repo)

	var publicKey *rsa.PublicKey
	if cfg.PublicKeyPath != "" {
		publicKey, err = crypto.LoadPublicKey(cfg.PublicKeyPath)
		if err != nil {
			logger.Error("failed to load public key", zap.Error(err))
			return err
		}
		logger.Info("public key loaded successfully")
	}

	newClient := services.NewClient(cfg, publicKey)
	pool := services.NewWorkerPool(cfg)
	pool.Start()

	tickerPoll := time.NewTicker(cfg.PollInterval)
	tickerReport := time.NewTicker(cfg.ReportInterval)

	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received, waiting for operations to complete...")
			tickerPoll.Stop()
			tickerReport.Stop()
			pool.Stop()

			wg.Wait()
			logger.Info("all operations completed, shutting down gracefully")
			return nil

		case <-tickerPoll.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := collectService.UpdateAllMetrics(ctx); err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						logger.Debug("request cancelled")
						return
					}
					logger.Error("failed to update metrics", zap.Error(err))
					return
				}
			}()

		case <-tickerReport.C:
			pool.Submit(func() {
				if err := queryService.SendMetrics(ctx, newClient); err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						logger.Debug("request cancelled")
						return
					}
					logger.Error("failed to send metrics", zap.Error(err))
					return
				}
				logger.Info("metrics sent successfully")

				if err := collectService.ResetPollCount(); err != nil {
					logger.Error("failed to reset poll count", zap.Error(err))
				}
			})
		}
	}
}
