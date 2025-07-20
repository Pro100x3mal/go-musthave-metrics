package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/go-chi/chi/v5"
)

func (mh *MetricsHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if mType == "" || mName == "" || mValue == "" {
		http.NotFound(w, r)
		return
	}

	if err := mh.writer.UpdateMetricFromParams(mType, mName, mValue); err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidMetricValue):
			http.Error(w, "Invalid Metric Value", http.StatusBadRequest)
		case errors.Is(err, models.ErrUnsupportedMetricType):
			http.Error(w, "Unsupported Metric Type", http.StatusBadRequest)
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func (mh *MetricsHandler) UpdateJSONHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	var metric models.Metrics
	err := json.NewDecoder(r.Body).Decode(&metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if metric.ID == "" || metric.MType == "" {
		http.Error(w, "Missing required metric fields", http.StatusBadRequest)
		return
	}

	err = mh.writer.UpdateJSONMetricFromParams(&metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func (mh *MetricsHandler) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	mValue, err := mh.reader.GetMetricValue(mType, mName)
	if err != nil {
		if errors.Is(err, models.ErrMetricNotFound) {
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

func (mh *MetricsHandler) GetJSONMetricHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	var metric models.Metrics
	err := json.NewDecoder(r.Body).Decode(&metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if metric.ID == "" || metric.MType == "" {
		http.Error(w, "Missing required metric fields", http.StatusBadRequest)
		return
	}

	respMetric, err := mh.reader.GetJSONMetricValue(&metric)
	if err != nil {
		if errors.Is(err, models.ErrMetricNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, models.ErrUnsupportedMetricType) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(respMetric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (mh *MetricsHandler) ListAllMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	list := mh.reader.GetAllMetrics()

	keys := make([]string, 0, len(list))
	for name := range list {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString("<html><body><h1>Metrics</h1><ul>")
	for _, name := range keys {
		val := list[name]
		builder.WriteString(fmt.Sprintf("<li>%s: %s</li>\n", name, val))
	}
	builder.WriteString("</ul></body></html>")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, builder.String())
}
