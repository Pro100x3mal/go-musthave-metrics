package handlers

import (
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/middlewares"
)

func (r *router) initRoutes(log *infrastructure.Logger, mh *MetricsHandler) {
	r.Use(middlewares.WithLogging(log))

	r.Get("/", mh.ListAllMetricsHandler)
	r.Get("/value/{mType}/{mName}", mh.GetMetricHandler)
	r.Post("/update/{mType}/{mName}/{mValue}", mh.UpdateHandler)
}
