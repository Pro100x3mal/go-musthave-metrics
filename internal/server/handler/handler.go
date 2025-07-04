package handler

import (
	"errors"
	"net/http"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repository"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/service"
	"github.com/go-chi/chi/v5"
)

type MetricsUpdater interface {
	UpdateMetricFromParams(mType, mName, mValue string) error
	GetMetricValue(mType, mName string) (string, error)
}
type metricsHandler struct {
	updater MetricsUpdater
}

func newMetricsHandler(updater MetricsUpdater) *metricsHandler {
	return &metricsHandler{
		updater: updater,
	}
}

func (h *metricsHandler) UpdateMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if mType == "" || mName == "" || mValue == "" {
		http.NotFound(w, r)
		return
	}

	if err := h.updater.UpdateMetricFromParams(mType, mName, mValue); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidMetricValue):
			http.Error(w, "Invalid Metric Value", http.StatusBadRequest)
		case errors.Is(err, service.ErrUnsupportedMetricType):
			http.Error(w, "Unsupported Metric Type", http.StatusBadRequest)
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func (h *metricsHandler) GetMetricValueHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	mValue, err := h.updater.GetMetricValue(mType, mName)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(mValue))
}
