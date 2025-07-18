package services

import (
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
)

type MetricsRepositoryReader interface {
	GetGauge(mName string) (float64, error)
	GetCounter(mName string) (int64, error)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}

type MetricsRepositoryWriter interface {
	UpdateMetrics(metric *models.Metrics) error
}

type MetricsRepositoryInterface interface {
	MetricsRepositoryReader
	MetricsRepositoryWriter
}

type MetricsService struct {
	reader MetricsRepositoryReader
	writer MetricsRepositoryWriter
}

func NewMetricsService(repository MetricsRepositoryInterface) *MetricsService {
	return &MetricsService{
		reader: repository,
		writer: repository,
	}
}

func (ms *MetricsService) UpdateMetricFromParams(mType, mName, mValue string) error {
	var metric models.Metrics
	metric.ID = mName
	metric.MType = mType

	switch mType {
	case models.Gauge:
		value, err := strconv.ParseFloat(mValue, 64)
		if err != nil {
			return models.ErrInvalidMetricValue
		}
		metric.Value = &value
	case models.Counter:
		delta, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			return models.ErrInvalidMetricValue
		}
		metric.Delta = &delta
	default:
		return models.ErrUnsupportedMetricType
	}

	if err := ms.writer.UpdateMetrics(&metric); err != nil {
		return err
	}

	return nil
}

func (ms *MetricsService) GetMetricValue(mType, mName string) (string, error) {
	switch mType {
	case models.Gauge:
		value, err := ms.reader.GetGauge(mName)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case models.Counter:
		value, err := ms.reader.GetCounter(mName)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(value, 10), nil
	default:
		return "", models.ErrUnsupportedMetricType
	}
}

func (ms *MetricsService) GetAllMetrics() map[string]string {
	list := make(map[string]string)

	for name, value := range ms.reader.GetAllGauges() {
		list[name] = strconv.FormatFloat(value, 'f', -1, 64)
	}

	for name, value := range ms.reader.GetAllCounters() {
		list[name] = strconv.FormatInt(value, 10)
	}
	return list
}
