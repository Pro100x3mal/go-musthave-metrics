package handler

import (
	"net/http"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/config"
	"github.com/go-chi/chi/v5"
)

func newRouter(mh *metricsHandler) chi.Router {
	r := chi.NewRouter()
	r.Get("/", mh.ListAllMetricsHandler)
	r.Get("/value/{mType}/{mName}", mh.GetMetricValueHandler)
	r.Post("/update/{mType}/{mName}/{mValue}", mh.UpdateMetricsHandler)

	return r
}

func Serve(cfg config.ServerConfig, reader MetricsReader, writer MetricsWriter) error {
	h := newMetricsHandler(reader, writer)
	router := newRouter(h)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}
	return srv.ListenAndServe()
}
