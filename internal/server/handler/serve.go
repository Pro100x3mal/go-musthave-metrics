package handler

import (
	"net/http"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/config"
	"github.com/go-chi/chi/v5"
)

func newRouter(rh *metricsReceiverHandler, qh *metricsQueryHandler) chi.Router {
	r := chi.NewRouter()

	r.Get("/", qh.ListAllMetricsHandler)
	r.Get("/value/{mType}/{mName}", qh.GetMetricHandler)
	r.Post("/update/{mType}/{mName}/{mValue}", rh.UpdateHandler)

	return r
}

func Serve(cfg config.ServerConfig, receiverService MetricsWriter, queryService MetricsReader) error {
	rh := newMetricsUpdateHandler(receiverService)
	qh := newMetricsQueryHandler(queryService)
	router := newRouter(rh, qh)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}
	return srv.ListenAndServe()
}
