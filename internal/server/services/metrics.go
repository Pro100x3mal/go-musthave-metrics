package services

import (
	"context"
	"errors"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
)

type MetricsRepositoryPinger interface {
	Ping(ctx context.Context) error
}

type MetricsService struct {
	reader repositories.RepositoryReader
	writer repositories.RepositoryWriter
	pinger MetricsRepositoryPinger
}

func NewMetricsService(repository repositories.Repository) *MetricsService {
	ms := &MetricsService{
		reader: repository,
		writer: repository,
	}

	if p, ok := repository.(MetricsRepositoryPinger); ok {
		ms.pinger = p
	}
	return ms
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
		return ms.writer.UpdateGauge(&metric)
	case models.Counter:
		delta, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			return models.ErrInvalidMetricValue
		}
		metric.Delta = &delta
		return ms.writer.UpdateCounter(&metric)
	default:
		return models.ErrUnsupportedMetricType
	}
}

func (ms *MetricsService) UpdateJSONMetricFromParams(metric *models.Metrics) error {
	if metric == nil {
		return models.ErrMetricNotFound
	}

	switch metric.MType {
	case models.Gauge:
		return ms.writer.UpdateGauge(metric)
	case models.Counter:
		return ms.writer.UpdateCounter(metric)
	default:
		return models.ErrUnsupportedMetricType
	}
}

func (ms *MetricsService) GetMetricValue(mType,
	mName string) (string, error) {
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

func (ms *MetricsService) GetJSONMetricValue(metric *models.Metrics) (*models.Metrics, error) {
	if metric == nil {
		return nil, models.ErrMetricNotFound
	}

	switch metric.MType {
	case models.Gauge:
		value, err := ms.reader.GetGauge(metric.ID)
		if err != nil {
			return nil, err
		}
		metric.Value = &value
		return metric, nil
	case models.Counter:
		delta, err := ms.reader.GetCounter(metric.ID)
		if err != nil {
			return nil, err
		}
		metric.Delta = &delta
		return metric, nil
	default:
		return nil, models.ErrUnsupportedMetricType
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

func (ms *MetricsService) PingCheck(ctx context.Context) error {
	if ms.pinger == nil {
		return errors.New("pinging not supported by this repository")
	}
	return ms.pinger.Ping(ctx)
}
