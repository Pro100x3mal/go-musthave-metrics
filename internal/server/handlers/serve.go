package handlers

import (
	"net/http"

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

func StartServer(cfg *configs.ServerConfig, log *infrastructure.Logger, mh *MetricsHandler) error {
	r := newRouter()
	r.initRoutes(log, mh)

	srv := newServer(cfg)
	srv.Handler = r

	log.Info("starting server...", zap.String("address", cfg.ServerAddr))

	if err := srv.ListenAndServe(); err != nil {
		log.Error("server failed", zap.Error(err))
		return err
	}

	return nil
}
