package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/services"
	"github.com/go-chi/chi/v5"
)

type MetricsReader interface {
	GetMetricValue(mType, mName string) (string, error)
	GetAllMetrics() map[string]string
}

type MetricsWriter interface {
	UpdateMetricFromParams(mType, mName, mValue string) error
}
type metricsQueryHandler struct {
	reader MetricsReader
}

type metricsReceiverHandler struct {
	writer MetricsWriter
}

func newMetricsUpdateHandler(writer MetricsWriter) *metricsReceiverHandler {
	return &metricsReceiverHandler{
		writer: writer,
	}
}

func newMetricsQueryHandler(reader MetricsReader) *metricsQueryHandler {
	return &metricsQueryHandler{
		reader: reader,
	}
}

func (rh *metricsReceiverHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if mType == "" || mName == "" || mValue == "" {
		http.NotFound(w, r)
		return
	}

	if err := rh.writer.UpdateMetricFromParams(mType, mName, mValue); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidMetricValue):
			http.Error(w, "Invalid Metric Value", http.StatusBadRequest)
		case errors.Is(err, services.ErrUnsupportedMetricType):
			http.Error(w, "Unsupported Metric Type", http.StatusBadRequest)
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func (qh *metricsQueryHandler) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	mValue, err := qh.reader.GetMetricValue(mType, mName)
	if err != nil {
		if errors.Is(err, repositories.ErrMetricNotFound) {
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

func (qh *metricsQueryHandler) ListAllMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	list := qh.reader.GetAllMetrics()

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
