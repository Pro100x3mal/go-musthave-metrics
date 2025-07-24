package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure"
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

type router struct {
	*chi.Mux
}

func newRouter() *router {
	return &router{
		chi.NewRouter(),
	}
}

type server struct {
	*http.Server
}

func newServer(cfg *configs.ServerConfig) *server {
	return &server{
		&http.Server{
			Addr: cfg.ServerAddr,
		},
	}
}

func StartServer(ctx context.Context, cfg *configs.ServerConfig, log *infrastructure.Logger, mh *MetricsHandler) error {
	r := newRouter()
	r.initRoutes(log, mh)

	srv := newServer(cfg)
	srv.Handler = r

	log.Info("starting server...", zap.String("address", cfg.ServerAddr))

	serverErrCh := make(chan error, 1)

	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- nil
			return
		}
		log.Error("unexpected server error", zap.Error(err))
		serverErrCh <- err
	}()

	select {
	case <-ctx.Done():
		log.Info("server is shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to shutdown server", zap.Error(err))
			return err
		}

		log.Info("server shutdown complete")
		return <-serverErrCh
	case err := <-serverErrCh:
		return err
	}
}
