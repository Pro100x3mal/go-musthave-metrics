package handlers

import (
	"context"
	"crypto/rsa"
	"errors"
	"html/template"
	"net/http"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/audit"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// MetricsServiceReader provides read-only operations for metrics retrieval.
type MetricsServiceReader interface {
	GetMetricValue(ctx context.Context, mType, mName string) (string, error)
	GetJSONMetricValue(ctx context.Context, metric *models.Metrics) (*models.Metrics, error)
	GetAllMetrics(ctx context.Context) (map[string]string, error)
}

// MetricsServiceWriter provides write operations for metrics updates.
type MetricsServiceWriter interface {
	UpdateMetricFromParams(ctx context.Context, mType, mName, mValue string) error
	UpdateJSONMetric(ctx context.Context, metric *models.Metrics) error
	UpdateJSONMetrics(ctx context.Context, metrics []models.Metrics) error
}

// MetricsServicePinger provides health check functionality for the underlying storage.
type MetricsServicePinger interface {
	PingCheck(ctx context.Context) error
}

// MetricsServiceInterface combines all metrics service capabilities.
type MetricsServiceInterface interface {
	MetricsServiceReader
	MetricsServiceWriter
	MetricsServicePinger
}

// MetricsHandler handles HTTP requests for metrics operations.
type MetricsHandler struct {
	reader       MetricsServiceReader
	writer       MetricsServiceWriter
	pinger       MetricsServicePinger
	logger       *zap.Logger
	cfg          *configs.ServerConfig
	auditManager audit.Publisher
	tmpl         *template.Template
	privateKey   *rsa.PrivateKey
}

// NewMetricsHandler creates a new MetricsHandler with the provided service, logger, configuration and audit manager.
func NewMetricsHandler(service MetricsServiceInterface, logger *zap.Logger, cfg *configs.ServerConfig, auditManager audit.Publisher, privateKey *rsa.PrivateKey) *MetricsHandler {
	mh := &MetricsHandler{
		reader:       service,
		writer:       service,
		logger:       logger,
		cfg:          cfg,
		auditManager: auditManager,
		tmpl:         template.Must(template.New("metrics").Parse(metricsTemplate)),
		privateKey:   privateKey,
	}

	if p, ok := service.(MetricsServicePinger); ok {
		mh.pinger = p
	}
	return mh
}

// StartServer starts the HTTP server and blocks until the context is cancelled or an error occurs.
// It gracefully shuts down the server when the context is cancelled.
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
