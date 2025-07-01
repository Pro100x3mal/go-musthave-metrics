package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/service"
)

type MetricsUpdater interface {
	UpdateMetricFromParams(mType, mName, mValue string) error
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

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[2] == "" || parts[3] == "" {
		http.NotFound(w, r)
		return
	}

	mType, mName, mValue := parts[1], parts[2], parts[3]
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
