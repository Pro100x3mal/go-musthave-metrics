package handler

import (
	"net/http"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/config"
)

func newRouter(mh *metricsHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/", mh.UpdateMetricsHandler)
	return mux
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
