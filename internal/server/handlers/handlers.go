package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/audit"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const metricsTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Metrics</title>
</head>
<body>
    <h1>Metrics</h1>
    <ul>
    {{range .}}
        <li>{{.Name}}: {{.Value}}</li>
    {{end}}
    </ul>
</body>
</html>`

// UpdateHandler handles metric updates via URL parameters.
// It accepts HTTP POST requests with the following URL pattern:
// POST /update/{mType}/{mName}/{mValue}
// where mType is either "gauge" or "counter", mName is the metric name, and mValue is the metric value.
// Returns 200 OK on success, 400 Bad Request for invalid metric types or values, 500 Internal Server Error on failure.
func (mh *MetricsHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if err := mh.writer.UpdateMetricFromParams(r.Context(), mType, mName, mValue); err != nil {
		mh.writeError(w, err, "failed to update metric")
		return
	}

	if mh.auditManager != nil && mh.auditManager.HasObservers() {
		ipAddress := audit.GetIPAddress(r)
		metric := &models.Metrics{ID: mName, MType: mType}
		auditEvent := audit.NewAuditEventFromMetric(metric, ipAddress)
		go mh.auditManager.NotifyAll(r.Context(), auditEvent)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// UpdateJSONHandler handles metric updates via JSON payload.
// It accepts HTTP POST requests with Content-Type: application/json and a JSON body containing a Metrics object.
// The JSON structure should include "id" (metric name), "type" (gauge or counter), and either "value" (for gauge) or "delta" (for counter).
// Returns 200 OK on success, 400 Bad Request for invalid data, 415 Unsupported Media Type for non-JSON content, 500 Internal Server Error on failure.
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

	err = mh.writer.UpdateJSONMetric(r.Context(), &metric)
	if err != nil {
		mh.writeError(w, err, "failed to update metric")
		return
	}

	if mh.auditManager != nil && mh.auditManager.HasObservers() {
		ipAddress := audit.GetIPAddress(r)
		auditEvent := audit.NewAuditEventFromMetric(&metric, ipAddress)
		go mh.auditManager.NotifyAll(r.Context(), auditEvent)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// UpdateBatchJSONHandler handles batch metric updates via JSON payload.
// It accepts HTTP POST requests with Content-Type: application/json and a JSON array of Metrics objects.
// Each metric in the array should include "id" (metric name), "type" (gauge or counter), and either "value" (for gauge) or "delta" (for counter).
// Returns 200 OK on success, 400 Bad Request for invalid data, 415 Unsupported Media Type for non-JSON content, 500 Internal Server Error on failure.
func (mh *MetricsHandler) UpdateBatchJSONHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	var metrics []models.Metrics
	err := json.NewDecoder(r.Body).Decode(&metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, metric := range metrics {
		if metric.ID == "" || metric.MType == "" {
			http.Error(w, "Missing required metric fields", http.StatusBadRequest)
			return
		}
	}

	err = mh.writer.UpdateJSONMetrics(r.Context(), metrics)
	if err != nil {
		mh.writeError(w, err, "failed to update metrics")
		return
	}

	if mh.auditManager != nil && mh.auditManager.HasObservers() {
		ipAddress := audit.GetIPAddress(r)
		auditEvent := audit.NewAuditEventFromMetrics(metrics, ipAddress)
		go mh.auditManager.NotifyAll(r.Context(), auditEvent)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// GetMetricHandler retrieves a single metric value via URL parameters.
// It accepts HTTP GET requests with the following URL pattern:
// GET /value/{mType}/{mName}
// where mType is either "gauge" or "counter", and mName is the metric name.
// Returns the metric value as plain text on success, 400 Bad Request for invalid metric types, 404 Not Found if metric doesn't exist, 500 Internal Server Error on failure.
func (mh *MetricsHandler) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	mValue, err := mh.reader.GetMetricValue(r.Context(), mType, mName)
	if err != nil {
		mh.writeError(w, err, "failed to get metric")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(mValue))
	if err != nil {
		mh.logger.Error("failed to write response", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// GetJSONMetricHandler retrieves a single metric value via JSON payload.
// It accepts HTTP POST requests with Content-Type: application/json and a JSON body containing a Metrics object with "id" and "type" fields.
// Returns a JSON response with the complete metric information including the current value.
// Returns 200 OK with JSON on success, 400 Bad Request for invalid data, 404 Not Found if metric doesn't exist, 415 Unsupported Media Type for non-JSON content, 500 Internal Server Error on failure.
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

	respMetric, err := mh.reader.GetJSONMetricValue(r.Context(), &metric)
	if err != nil {
		mh.writeError(w, err, "failed to get metric")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(respMetric)
	if err != nil {
		mh.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

type metricItem struct {
	Name  string
	Value string
}

// ListAllMetricsHandler returns an HTML page with all stored metrics.
// It accepts HTTP GET requests to the root path "/" and returns an HTML page with a list of all metrics and their values.
// Returns 200 OK with HTML content on success, 500 Internal Server Error on failure.
func (mh *MetricsHandler) ListAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	list, err := mh.reader.GetAllMetrics(r.Context())
	if err != nil {
		mh.writeError(w, err, "failed to get metrics")
		return
	}

	keys := make([]string, 0, len(list))
	for name := range list {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	items := make([]metricItem, 0, len(keys))
	for _, name := range keys {
		items = append(items, metricItem{
			Name:  name,
			Value: list[name],
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	err = mh.tmpl.Execute(w, items)
	if err != nil {
		mh.logger.Error("failed to execute template", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// PingDBHandler checks the database connection health.
// It accepts HTTP GET requests to the "/ping" endpoint and verifies that the database connection is alive.
// Returns 200 OK if the database connection is healthy, 501 Not Implemented if database storage is not configured, 500 Internal Server Error if the connection check fails.
func (mh *MetricsHandler) PingDBHandler(w http.ResponseWriter, r *http.Request) {
	if mh.pinger == nil {
		mh.logger.Error("database connection check functionality not implemented for current storage type")
		http.Error(w, "Database connection check functionality not implemented for current storage type", http.StatusNotImplemented)
		return
	}

	if err := mh.pinger.PingCheck(r.Context()); err != nil {
		mh.logger.Error("database connection check failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (mh *MetricsHandler) writeError(w http.ResponseWriter, err error, internalErrorMessage string) {
	switch {
	case errors.Is(err, context.Canceled):
		mh.logger.Debug("request canceled by client")
		return

	case errors.Is(err, context.DeadlineExceeded):
		http.Error(w, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
		return

	case errors.Is(err, models.ErrUnsupportedMetricType):
		http.Error(w, models.ErrUnsupportedMetricType.Error(), http.StatusBadRequest)
		return

	case errors.Is(err, models.ErrInvalidMetricValue):
		http.Error(w, models.ErrInvalidMetricValue.Error(), http.StatusBadRequest)
		return

	case errors.Is(err, models.ErrMetricNotFound):
		http.Error(w, models.ErrMetricNotFound.Error(), http.StatusNotFound)
		return

	default:
		mh.logger.Error(internalErrorMessage, zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
