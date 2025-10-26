package models

import "errors"

const (
	// Counter represents the counter metric type.
	Counter = "counter"
	// Gauge represents the gauge metric type.
	Gauge = "gauge"
)

var (
	// ErrMetricNotFound is returned when a requested metric does not exist.
	ErrMetricNotFound = errors.New("metric not found")
	// ErrInvalidMetricValue is returned when a metric value cannot be parsed or is invalid.
	ErrInvalidMetricValue = errors.New("invalid metric value")
	// ErrUnsupportedMetricType is returned when an unsupported metric type is used.
	ErrUnsupportedMetricType = errors.New("unsupported metric type")
)

// Metrics represents a single metric with its type and value.
// For counter metrics, Delta field is used. For gauge metrics, Value field is used.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}
