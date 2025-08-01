package handlers

import (
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/middlewares"
	"github.com/go-chi/chi/v5"
)

func initRoutes(r *chi.Mux, mh *MetricsHandler, db *DBHandler) {
	r.Use(middlewares.NewLoggerHandler(mh.logger).Middleware)
	r.Use(middlewares.NewCompressHandler(mh.logger).Middleware)

	r.Get("/", mh.ListAllMetricsHandler)
	if db != nil {
		r.Get("/ping", db.PingDBHandler)
	}
	r.Route("/value", func(r chi.Router) {
		r.Post("/", mh.GetJSONMetricHandler)
		r.Get("/{mType}/{mName}", mh.GetMetricHandler)
	})
	r.Route("/update", func(r chi.Router) {
		r.Post("/", mh.UpdateJSONHandler)
		r.Post("/{mType}/{mName}/{mValue}", mh.UpdateHandler)
	})
}
