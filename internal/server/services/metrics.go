package services

import (
	"errors"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
)

type RepositoryReader interface {
	GetGauge(mName string) (float64, error)
	GetCounter(mName string) (int64, error)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}

type RepositoryWriter interface {
	UpdateMetrics(metric *models.Metrics) error
}
type MetricsReceiverService struct {
	writer RepositoryWriter
}

type MetricsQueryService struct {
	reader RepositoryReader
}

func NewMetricsReceiverService(writer RepositoryWriter) *MetricsReceiverService {
	return &MetricsReceiverService{
		writer: writer,
	}
}

func NewMetricsQueryService(reader RepositoryReader) *MetricsQueryService {
	return &MetricsQueryService{
		reader: reader,
	}
}

var (
	ErrInvalidMetricValue    = errors.New("invalid metric value")
	ErrUnsupportedMetricType = errors.New("unsupported metric type")
)

func (rs *MetricsReceiverService) UpdateMetricFromParams(mType, mName, mValue string) error {
	var metric models.Metrics
	metric.ID = mName
	metric.MType = mType

	switch mType {
	case models.Gauge:
		value, err := strconv.ParseFloat(mValue, 64)
		if err != nil {
			return ErrInvalidMetricValue
		}
		metric.Value = &value
	case models.Counter:
		delta, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			return ErrInvalidMetricValue
		}
		metric.Delta = &delta
	default:
		return ErrUnsupportedMetricType
	}

	if err := rs.writer.UpdateMetrics(&metric); err != nil {
		return err
	}

	return nil
}

func (qs *MetricsQueryService) GetMetricValue(mType, mName string) (string, error) {
	switch mType {
	case models.Gauge:
		value, err := qs.reader.GetGauge(mName)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case models.Counter:
		value, err := qs.reader.GetCounter(mName)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(value, 10), nil
	default:
		return "", ErrUnsupportedMetricType
	}
}

func (qs *MetricsQueryService) GetAllMetrics() map[string]string {
	list := make(map[string]string)

	for name, value := range qs.reader.GetAllGauges() {
		list[name] = strconv.FormatFloat(value, 'f', -1, 64)
	}

	for name, value := range qs.reader.GetAllCounters() {
		list[name] = strconv.FormatInt(value, 10)
	}
	return list
}
