package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/model"
)

type MetricsUpdater interface {
	UpdateMetrics(m *model.Metrics) error
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
	if len(parts) != 4 || parts[2] == "" {
		http.NotFound(w, r)
		return
	}

	mType, mName, mValue := parts[1], parts[2], parts[3]

	var metric model.Metrics
	metric.ID = mName
	metric.MType = mType

	switch mType {
	case model.Gauge:
		value, err := strconv.ParseFloat(mValue, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid value for %v metric: %s", model.Gauge, mValue), http.StatusBadRequest)
			return
		}
		metric.Value = &value
	case model.Counter:
		delta, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid value for %v metric: %s", model.Counter, mValue), http.StatusBadRequest)
			return
		}
		metric.Delta = &delta
	default:
		http.Error(w, fmt.Sprintf("Unsupported metric type: %s", mType), http.StatusBadRequest)
	}

	if err := h.updater.UpdateMetrics(&metric); err != nil {
		http.Error(w, fmt.Sprintf("Update metric failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}
