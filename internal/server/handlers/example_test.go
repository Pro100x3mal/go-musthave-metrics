package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/logger"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/services"
	"github.com/go-chi/chi/v5"
)

// setupTestServer creates a test server with a real metrics handler
func setupTestServer() *httptest.Server {
	// Create in-memory storage
	repo := repositories.NewMemStorage()

	// Create service
	service := services.NewMetricsService(repo)

	// Create config
	cfg := &configs.ServerConfig{
		ServerAddr: "localhost:8080",
		LogLevel:   "info",
	}

	// Create logger
	log, _ := logger.NewLogger(cfg)

	// Create handler
	handler := handlers.NewMetricsHandler(service, log, cfg, nil)

	// Setup router
	r := chi.NewRouter()
	r.Post("/update/{mType}/{mName}/{mValue}", handler.UpdateHandler)
	r.Post("/update/", handler.UpdateJSONHandler)
	r.Post("/updates/", handler.UpdateBatchJSONHandler)
	r.Get("/value/{mType}/{mName}", handler.GetMetricHandler)
	r.Post("/value/", handler.GetJSONMetricHandler)
	r.Get("/", handler.ListAllMetricsHandler)
	r.Get("/ping", handler.PingDBHandler)

	return httptest.NewServer(r)
}

// Example_updateGaugeViaURL demonstrates updating a gauge metric using URL parameters.
func Example_updateGaugeViaURL() {
	ts := setupTestServer()
	defer ts.Close()

	// Send POST request to update gauge metric
	resp, err := http.Post(ts.URL+"/update/gauge/temperature/23.5", "text/plain", nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Request: POST /update/gauge/temperature/23.5")

	// Output:
	// Status: 200
	// Request: POST /update/gauge/temperature/23.5
}

// Example_updateCounterViaURL demonstrates updating a counter metric using URL parameters.
func Example_updateCounterViaURL() {
	ts := setupTestServer()
	defer ts.Close()

	// Send POST request to update counter metric
	resp, err := http.Post(ts.URL+"/update/counter/requests/5", "text/plain", nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Request: POST /update/counter/requests/5")

	// Output:
	// Status: 200
	// Request: POST /update/counter/requests/5
}

// Example_updateGaugeJSON demonstrates updating a gauge metric using JSON.
func Example_updateGaugeJSON() {
	ts := setupTestServer()
	defer ts.Close()

	// Prepare JSON payload
	value := 23.5
	metric := models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
		Value: &value,
	}
	body, _ := json.Marshal(metric)

	// Send POST request with JSON
	resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Printf("Request body: %s\n", body)

	// Output:
	// Status: 200
	// Request body: {"id":"temperature","type":"gauge","value":23.5}
}

// Example_updateCounterJSON demonstrates updating a counter metric using JSON.
func Example_updateCounterJSON() {
	ts := setupTestServer()
	defer ts.Close()

	// Prepare JSON payload
	delta := int64(10)
	metric := models.Metrics{
		ID:    "requests",
		MType: models.Counter,
		Delta: &delta,
	}
	body, _ := json.Marshal(metric)

	// Send POST request with JSON
	resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Printf("Request body: %s\n", body)

	// Output:
	// Status: 200
	// Request body: {"id":"requests","type":"counter","delta":10}
}

// Example_batchUpdate demonstrates updating multiple metrics in a single request.
func Example_batchUpdate() {
	ts := setupTestServer()
	defer ts.Close()

	// Prepare batch of metrics
	value := 23.5
	delta := int64(10)
	metrics := []models.Metrics{
		{
			ID:    "temperature",
			MType: models.Gauge,
			Value: &value,
		},
		{
			ID:    "requests",
			MType: models.Counter,
			Delta: &delta,
		},
	}
	body, _ := json.Marshal(metrics)

	// Send POST request with JSON array
	resp, err := http.Post(ts.URL+"/updates/", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Metrics updated:", len(metrics))

	// Output:
	// Status: 200
	// Metrics updated: 2
}

// Example_getMetricViaURL demonstrates retrieving a metric value using URL parameters.
func Example_getMetricViaURL() {
	ts := setupTestServer()
	defer ts.Close()

	// First, update a metric
	updateResp, err := http.Post(ts.URL+"/update/gauge/temperature/23.5", "text/plain", nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	updateResp.Body.Close()

	// Then retrieve it
	resp, err := http.Get(ts.URL + "/value/gauge/temperature")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Value:", string(body))

	// Output:
	// Status: 200
	// Value: 23.5
}

// Example_getMetricJSON demonstrates retrieving a metric value using JSON.
func Example_getMetricJSON() {
	ts := setupTestServer()
	defer ts.Close()

	// First, update a metric
	value := 23.5
	updateMetric := models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
		Value: &value,
	}
	updateBody, _ := json.Marshal(updateMetric)
	updateResp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewReader(updateBody))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	updateResp.Body.Close()

	// Then retrieve it
	getMetric := models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
	}
	getBody, _ := json.Marshal(getMetric)

	resp, err := http.Post(ts.URL+"/value/", "application/json", bytes.NewReader(getBody))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	var result models.Metrics
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Println("Status:", resp.StatusCode)
	fmt.Printf("Metric ID: %s, Type: %s, Value: %.1f\n", result.ID, result.MType, *result.Value)

	// Output:
	// Status: 200
	// Metric ID: temperature, Type: gauge, Value: 23.5
}

// Example_listAllMetrics demonstrates retrieving all metrics as HTML.
func Example_listAllMetrics() {
	ts := setupTestServer()
	defer ts.Close()

	// Update some metrics
	resp1, err := http.Post(ts.URL+"/update/gauge/temperature/23.5", "text/plain", nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	resp1.Body.Close()

	resp2, err := http.Post(ts.URL+"/update/counter/requests/10", "text/plain", nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	resp2.Body.Close()

	// Get all metrics
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Content-Type:", resp.Header.Get("Content-Type"))

	// Output:
	// Status: 200
	// Content-Type: text/html; charset=utf-8
}
