package service

import (
	"errors"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/model"
)

var (
	ErrUnsupportedMetricValue = errors.New("unsupported metric value")
	ErrUnsupportedMetricType  = errors.New("unsupported metric type")
)

type MetricsRepository interface {
	UpdateGauge(id string, value float64)
	UpdateCounter(id string, delta int64)
}
type MetricsService struct {
	repo MetricsRepository
}

func NewMetricsService(repo MetricsRepository) *MetricsService {
	return &MetricsService{
		repo: repo,
	}
}

func (ms *MetricsService) UpdateMetrics(m *model.Metrics) error {
	switch m.MType {
	case model.Gauge:
		if m.Value == nil {
			return ErrUnsupportedMetricValue
		}
		ms.repo.UpdateGauge(m.ID, *m.Value)
	case model.Counter:
		if m.Delta == nil {
			return ErrUnsupportedMetricValue
		}
		ms.repo.UpdateCounter(m.ID, *m.Delta)
	default:
		return ErrUnsupportedMetricType

	}
	return nil
}
