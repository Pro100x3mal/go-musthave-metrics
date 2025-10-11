package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/audit"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type MetricsServiceReader interface {
	GetMetricValue(ctx context.Context, mType, mName string) (string, error)
	GetJSONMetricValue(ctx context.Context, metric *models.Metrics) (*models.Metrics, error)
	GetAllMetrics(ctx context.Context) (map[string]string, error)
}

type MetricsServiceWriter interface {
	UpdateMetricFromParams(ctx context.Context, mType, mName, mValue string) error
	UpdateJSONMetric(ctx context.Context, metric *models.Metrics) error
	UpdateJSONMetrics(ctx context.Context, metrics []models.Metrics) error
}

type MetricsServicePinger interface {
	PingCheck(ctx context.Context) error
}

type MetricsServiceInterface interface {
	MetricsServiceReader
	MetricsServiceWriter
	MetricsServicePinger
}

type MetricsHandler struct {
	reader       MetricsServiceReader
	writer       MetricsServiceWriter
	pinger       MetricsServicePinger
	logger       *zap.Logger
	cfg          *configs.ServerConfig
	auditManager *audit.AuditManager
}

func NewMetricsHandler(service MetricsServiceInterface, logger *zap.Logger, cfg *configs.ServerConfig, auditManager *audit.AuditManager) *MetricsHandler {
	mh := &MetricsHandler{
		reader:       service,
		writer:       service,
		logger:       logger,
		cfg:          cfg,
		auditManager: auditManager,
	}

	if p, ok := service.(MetricsServicePinger); ok {
		mh.pinger = p
	}
	return mh
}

func (mh *MetricsHandler) StartServer(ctx context.Context) error {
	r := chi.NewRouter()
	initRoutes(r, mh)

	srv := &http.Server{
		Addr:    mh.cfg.ServerAddr,
		Handler: r,
	}

	mh.logger.Info("starting server...", zap.String("address", mh.cfg.ServerAddr))

	serverErrCh := make(chan error, 1)

	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- nil
			return
		}

		mh.logger.Error("unexpected server error", zap.Error(err))
		serverErrCh <- err
	}()

	select {
	case <-ctx.Done():
		mh.logger.Info("server is shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			mh.logger.Error("failed to shutdown server", zap.Error(err))
			return err
		}

		mh.logger.Info("server shutdown complete")
		return nil
	case err := <-serverErrCh:
		return err
	}
}
