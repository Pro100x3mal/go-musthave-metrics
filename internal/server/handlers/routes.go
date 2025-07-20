package handlers

import (
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/middlewares"
	"github.com/go-chi/chi/v5"
)

func (r *router) initRoutes(log *infrastructure.Logger, mh *MetricsHandler) {
	r.Use(middlewares.WithLogging(log))

	r.Get("/", mh.ListAllMetricsHandler)
	r.Route("/value", func(r chi.Router) {
		r.Post("/", mh.GetJSONMetricHandler)
		r.Get("/{mType}/{mName}", mh.GetMetricHandler)
	})
	r.Route("/update", func(r chi.Router) {
		r.Post("/", mh.UpdateJSONHandler)
		r.Post("/{mType}/{mName}/{mValue}", mh.UpdateHandler)
	})
}
