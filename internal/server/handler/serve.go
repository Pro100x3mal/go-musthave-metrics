package handler

import (
	"net/http"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/config"
	"github.com/go-chi/chi/v5"
)

func newRouter(mh *metricsHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/update/{mType}/{mName}/{mValue}", mh.UpdateMetricsHandler)
	r.Get("/value/{mType}/{mName}", mh.GetMetricValueHandler)

	return r
}

func Serve(cfg config.ServerConfig, updater MetricsUpdater) error {
	h := newMetricsHandler(updater)
	router := newRouter(h)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}
	return srv.ListenAndServe()
}
