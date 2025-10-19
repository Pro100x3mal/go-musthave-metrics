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
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/audit"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/logger"
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

	zLog, err := logger.NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer zLog.Sync()

	mainLogger := zLog.Named("main")
	srvLogger := zLog.Named("server")

	mainLogger.Info("starting application")

	var repo repositories.Repository
	var wg sync.WaitGroup

	switch {
	case cfg.DatabaseDSN != "":
		dbLogger := zLog.Named("database")
		dbRepo, err := repositories.NewDB(ctx, cfg, dbLogger)
		if err != nil {
			dbLogger.Error("failed to initialize database storage", zap.Error(err))
			return err
		}
		defer dbRepo.Close()
		repo = retry.NewRepoWithRetry(dbRepo, []time.Duration{}, 0)
	case cfg.FileStoragePath != "":
		fsLogger := zLog.Named("file_storage")
		msRepo := repositories.NewMemStorage()
		repo, err = repositories.NewFileStorage(ctx, cfg, msRepo, &wg, fsLogger)
		if err != nil {
			fsLogger.Error("failed to initialize file storage", zap.Error(err))
			return err
		}
	default:
		msLogger := zLog.Named("memory_storage")
		msLogger.Info("initializing in-memory storage")
		repo = repositories.NewMemStorage()
		msLogger.Info("in-memory storage initialized successfully")
	}

	service := services.NewMetricsService(repo)

	auditLogger := zLog.Named("audit")
	auditManager := audit.NewAuditManager(auditLogger)

	if cfg.AuditFile != "" {
		fileObserver := audit.NewFileAuditObserver(cfg.AuditFile)
		auditManager.Attach(fileObserver)
		auditLogger.Info("file audit observer enabled", zap.String("file", cfg.AuditFile))
	}

	if cfg.AuditURL != "" {
		httpObserver := audit.NewHTTPAuditObserver(cfg.AuditURL)
		auditManager.Attach(httpObserver)
		auditLogger.Info("HTTP audit observer enabled", zap.String("url", cfg.AuditURL))
	}

	handler := handlers.NewMetricsHandler(service, srvLogger, cfg, auditManager)

	if err = handler.StartServer(ctx); err != nil {
		srvLogger.Error("server failed", zap.Error(err))
	}

	wg.Wait()
	mainLogger.Info("application stopped gracefully")
	return err
}
