package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/logger"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type MetricsServiceReader interface {
	GetMetricValue(mType, mName string) (string, error)
	GetJSONMetricValue(metric *models.Metrics) (*models.Metrics, error)
	GetAllMetrics() map[string]string
}

type MetricsServiceWriter interface {
	UpdateMetricFromParams(mType, mName, mValue string) error
	UpdateJSONMetricFromParams(metric *models.Metrics) error
}

type MetricsServiceInterface interface {
	MetricsServiceReader
	MetricsServiceWriter
}

type MetricsHandler struct {
	reader MetricsServiceReader
	writer MetricsServiceWriter
}

func NewMetricsHandler(service MetricsServiceInterface) *MetricsHandler {
	return &MetricsHandler{
		reader: service,
		writer: service,
	}
}

func StartServer(ctx context.Context, cfg *configs.ServerConfig, mh *MetricsHandler) error {
	r := chi.NewRouter()
	initRoutes(r, mh)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: r,
	}

	logger.Log.Info("starting server...", zap.String("address", cfg.ServerAddr))

	serverErrCh := make(chan error, 1)

	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- nil
			return
		}

		logger.Log.Error("unexpected server error", zap.Error(err))
		serverErrCh <- err
	}()

	select {
	case <-ctx.Done():
		logger.Log.Info("server is shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("failed to shutdown server", zap.Error(err))
			return err
		}

		logger.Log.Info("server shutdown complete")
		return nil
	case err := <-serverErrCh:
		return err
	}
}
