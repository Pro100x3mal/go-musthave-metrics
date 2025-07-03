package service

import (
	"errors"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/model"
)

type MetricsRepository interface {
	UpdateMetrics(metric *model.Metrics) error
}
type MetricsService struct {
	repo MetricsRepository
}

func NewMetricsService(repo MetricsRepository) *MetricsService {
	return &MetricsService{
		repo: repo,
	}
}

var (
	ErrInvalidMetricValue    = errors.New("invalid metric value")
	ErrUnsupportedMetricType = errors.New("unsupported metric type")
)

func (ms *MetricsService) UpdateMetricFromParams(mType, mName, mValue string) error {
	var metric model.Metrics
	metric.ID = mName
	metric.MType = mType

	switch mType {
	case model.Gauge:
		value, err := strconv.ParseFloat(mValue, 64)
		if err != nil {
			return ErrInvalidMetricValue
		}
		metric.Value = &value
	case model.Counter:
		delta, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			return ErrInvalidMetricValue
		}
		metric.Delta = &delta
	default:
		return ErrUnsupportedMetricType
	}

	if err := ms.repo.UpdateMetrics(&metric); err != nil {
		return err
	}

	return nil
}
