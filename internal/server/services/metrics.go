package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
)

// MetricsRepositoryPinger provides health check functionality for the metrics repository.
type MetricsRepositoryPinger interface {
	Ping(ctx context.Context) error
}

// MetricsService provides business logic for metrics operations.
type MetricsService struct {
	reader repositories.RepositoryReader
	writer repositories.RepositoryWriter
	pinger MetricsRepositoryPinger
}

// NewMetricsService creates a new MetricsService with the provided repository.
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

// UpdateMetricFromParams updates a metric using URL parameters.
// It parses the metric value according to its type and updates the repository.
func (ms *MetricsService) UpdateMetricFromParams(ctx context.Context, mType, mName, mValue string) error {
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
		return ms.writer.UpdateGauge(ctx, &metric)
	case models.Counter:
		delta, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			return models.ErrInvalidMetricValue
		}
		metric.Delta = &delta
		return ms.writer.UpdateCounter(ctx, &metric)
	default:
		return models.ErrUnsupportedMetricType
	}
}

// UpdateJSONMetric updates a single metric from a JSON request.
func (ms *MetricsService) UpdateJSONMetric(ctx context.Context, metric *models.Metrics) error {
	if metric == nil {
		return models.ErrMetricNotFound
	}

	switch metric.MType {
	case models.Gauge:
		return ms.writer.UpdateGauge(ctx, metric)
	case models.Counter:
		return ms.writer.UpdateCounter(ctx, metric)
	default:
		return models.ErrUnsupportedMetricType
	}
}

// UpdateJSONMetrics updates multiple metrics from a JSON request in a batch operation.
func (ms *MetricsService) UpdateJSONMetrics(ctx context.Context, metrics []models.Metrics) error {
	if metrics == nil {
		return models.ErrMetricNotFound
	}
	return ms.writer.UpdateMetrics(ctx, metrics)
}

// GetMetricValue retrieves a metric value as a string by its type and name.
func (ms *MetricsService) GetMetricValue(ctx context.Context, mType, mName string) (string, error) {
	switch mType {
	case models.Gauge:
		value, err := ms.reader.GetGauge(ctx, mName)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case models.Counter:
		value, err := ms.reader.GetCounter(ctx, mName)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(value, 10), nil
	default:
		return "", models.ErrUnsupportedMetricType
	}
}

// GetJSONMetricValue retrieves a metric value and returns it as a Metrics object.
func (ms *MetricsService) GetJSONMetricValue(ctx context.Context, metric *models.Metrics) (*models.Metrics, error) {
	if metric == nil {
		return nil, models.ErrMetricNotFound
	}

	switch metric.MType {
	case models.Gauge:
		value, err := ms.reader.GetGauge(ctx, metric.ID)
		if err != nil {
			return nil, err
		}
		metric.Value = &value
		return metric, nil
	case models.Counter:
		delta, err := ms.reader.GetCounter(ctx, metric.ID)
		if err != nil {
			return nil, err
		}
		metric.Delta = &delta
		return metric, nil
	default:
		return nil, models.ErrUnsupportedMetricType
	}
}

// GetAllMetrics retrieves all stored metrics as a map of name to value strings.
func (ms *MetricsService) GetAllMetrics(ctx context.Context) (map[string]string, error) {
	list := make(map[string]string)

	gauges, err := ms.reader.GetAllGauges(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	for name, value := range gauges {
		list[name] = strconv.FormatFloat(value, 'f', -1, 64)
	}

	counters, err := ms.reader.GetAllCounters(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	for name, delta := range counters {
		list[name] = strconv.FormatInt(delta, 10)
	}
	return list, nil
}

// PingCheck verifies the health of the underlying storage connection.
func (ms *MetricsService) PingCheck(ctx context.Context) error {
	if ms.pinger == nil {
		return errors.New("pinging not supported by this repository")
	}
	return ms.pinger.Ping(ctx)
}
