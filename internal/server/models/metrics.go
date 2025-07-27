package models

import "errors"

const (
	Counter = "counter"
	Gauge   = "gauge"
)

var (
	ErrMetricNotFound        = errors.New("metric not found")
	ErrInvalidMetricValue    = errors.New("invalid metric value")
	ErrUnsupportedMetricType = errors.New("unsupported metric type")
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}
