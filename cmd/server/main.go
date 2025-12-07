package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	grpcserver "github.com/Pro100x3mal/go-musthave-metrics/internal/server/grpc"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/audit"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/ipfilter"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/logger"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories/retry"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/services"
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
		return fmt.Errorf("failed to get config: %w", err)
	}

	zLog, err := logger.NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer zLog.Sync()

	mainLogger := zLog.Named("main")

	mainLogger.Info("starting application")

	var privateKey *rsa.PrivateKey
	if cfg.PrivateKeyPath != "" {
		privateKey, err = crypto.LoadPrivateKey(cfg.PrivateKeyPath)
		if err != nil {
			mainLogger.Error("failed to load private key", zap.Error(err))
			return err
		}
		mainLogger.Info("private key loaded successfully")
	}

	var trustedSubnet *net.IPNet
	if cfg.TrustedSubnet != "" {
		_, trustedSubnet, err = net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			mainLogger.Error("failed to parse trusted subnet", zap.Error(err), zap.String("subnet", cfg.TrustedSubnet))
			return fmt.Errorf("invalid trusted subnet: %w", err)
		}
		mainLogger.Info("trusted subnet configured", zap.String("subnet", cfg.TrustedSubnet))
	}

	ipFilterLogger := zLog.Named("ipfilter")
	ipFilterService := ipfilter.NewIPFilter(trustedSubnet, ipFilterLogger)

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
		fileObserver, err := audit.NewFileAuditObserver(cfg.AuditFile)
		if err != nil {
			auditLogger.Error("failed to initialize file audit observer", zap.Error(err))
			return err
		}
		defer func(fileObserver *audit.FileAuditObserver) {
			err := fileObserver.Close()
			if err != nil {
				auditLogger.Error("failed to close file audit observer", zap.Error(err))
			}
		}(fileObserver)
		auditManager.Attach(fileObserver)
		auditLogger.Info("file audit observer enabled", zap.String("file", cfg.AuditFile))
	}

	if cfg.AuditURL != "" {
		httpObserver := audit.NewHTTPAuditObserver(cfg.AuditURL)
		auditManager.Attach(httpObserver)
		auditLogger.Info("HTTP audit observer enabled", zap.String("url", cfg.AuditURL))
	}

	if cfg.GRPCAddr != "" {
		grpcLogger := zLog.Named("grpc")
		metricsServer := grpcserver.NewMetricsServer(service, grpcLogger)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := grpcserver.StartGRPCServer(ctx, cfg.GRPCAddr, ipFilterService, metricsServer, grpcLogger); err != nil {
				grpcLogger.Error("gRPC server failed", zap.Error(err))
			}
		}()
		mainLogger.Info("gRPC server started", zap.String("address", cfg.GRPCAddr))
	} else {
		srvLogger := zLog.Named("server")
		handler := handlers.NewMetricsHandler(service, srvLogger, cfg, auditManager, privateKey, ipFilterService)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := handler.StartServer(ctx); err != nil {
				srvLogger.Error("HTTP server failed", zap.Error(err))
			}
		}()
		mainLogger.Info("HTTP server started", zap.String("address", cfg.ServerAddr))
	}

	<-ctx.Done()
	mainLogger.Info("shutting down servers...")

	shutdownDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		mainLogger.Info("application stopped gracefully")
	case <-time.After(10 * time.Second):
		mainLogger.Warn("shutdown timeout exceeded, forcing exit")
	}
	return nil
}
