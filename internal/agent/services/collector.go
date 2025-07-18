package services

import (
	"fmt"
	"math/rand"
	"runtime"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
)

type RepositoryWriter interface {
	UpdateMetrics(metric *models.Metrics) error
	ResetMetricValue(metric *models.Metrics) error
}

type MetricsCollectService struct {
	writer  RepositoryWriter
	metrics *metricsProvider
}

func NewMetricsCollectService(writer RepositoryWriter) *MetricsCollectService {
	return &MetricsCollectService{
		writer:  writer,
		metrics: newMetricsProvider(),
	}
}

const (
	pollCountMetric   = "PollCount"
	randomValueMetric = "RandomValue"
)

type metricsProvider struct {
	stats          *runtime.MemStats
	runtimeMetrics map[string]func(m *runtime.MemStats) float64
}

func newMetricsProvider() *metricsProvider {
	return &metricsProvider{
		stats: &runtime.MemStats{},
		runtimeMetrics: map[string]func(m *runtime.MemStats) float64{
			"MetricAlloc":         func(m *runtime.MemStats) float64 { return float64(m.Alloc) },
			"MetricBuckHashSys":   func(m *runtime.MemStats) float64 { return float64(m.BuckHashSys) },
			"MetricFrees":         func(m *runtime.MemStats) float64 { return float64(m.Frees) },
			"MetricGCCPUFraction": func(m *runtime.MemStats) float64 { return m.GCCPUFraction },
			"MetricGCSys":         func(m *runtime.MemStats) float64 { return float64(m.GCSys) },
			"MetricHeapAlloc":     func(m *runtime.MemStats) float64 { return float64(m.HeapAlloc) },
			"MetricHeapIdle":      func(m *runtime.MemStats) float64 { return float64(m.HeapIdle) },
			"MetricHeapInuse":     func(m *runtime.MemStats) float64 { return float64(m.HeapInuse) },
			"MetricHeapObjects":   func(m *runtime.MemStats) float64 { return float64(m.HeapObjects) },
			"MetricHeapReleased":  func(m *runtime.MemStats) float64 { return float64(m.HeapReleased) },
			"MetricHeapSys":       func(m *runtime.MemStats) float64 { return float64(m.HeapSys) },
			"MetricLastGC":        func(m *runtime.MemStats) float64 { return float64(m.LastGC) },
			"MetricLookups":       func(m *runtime.MemStats) float64 { return float64(m.Lookups) },
			"MetricMCacheInuse":   func(m *runtime.MemStats) float64 { return float64(m.MCacheInuse) },
			"MetricMCacheSys":     func(m *runtime.MemStats) float64 { return float64(m.MCacheSys) },
			"MetricMSpanInuse":    func(m *runtime.MemStats) float64 { return float64(m.MSpanInuse) },
			"MetricMSpanSys":      func(m *runtime.MemStats) float64 { return float64(m.MSpanSys) },
			"MetricMallocs":       func(m *runtime.MemStats) float64 { return float64(m.Mallocs) },
			"MetricNextGC":        func(m *runtime.MemStats) float64 { return float64(m.NextGC) },
			"MetricNumForcedGC":   func(m *runtime.MemStats) float64 { return float64(m.NumForcedGC) },
			"MetricNumGC":         func(m *runtime.MemStats) float64 { return float64(m.NumGC) },
			"MetricOtherSys":      func(m *runtime.MemStats) float64 { return float64(m.OtherSys) },
			"MetricPauseTotalNs":  func(m *runtime.MemStats) float64 { return float64(m.PauseTotalNs) },
			"MetricStackInuse":    func(m *runtime.MemStats) float64 { return float64(m.StackInuse) },
			"MetricStackSys":      func(m *runtime.MemStats) float64 { return float64(m.StackSys) },
			"MetricSys":           func(m *runtime.MemStats) float64 { return float64(m.Sys) },
			"MetricTotalAlloc":    func(m *runtime.MemStats) float64 { return float64(m.TotalAlloc) },
		},
	}
}

func (cs *MetricsCollectService) updateCollectMetrics() error {
	runtime.ReadMemStats(cs.metrics.stats)

	for name, fn := range cs.metrics.runtimeMetrics {
		val := fn(cs.metrics.stats)
		err := cs.writer.UpdateMetrics(&models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &val,
		})
		if err != nil {
			return fmt.Errorf("update %s metric error: %w", name, err)

		}
	}

	return nil
}

func (cs *MetricsCollectService) updateRandomValue() error {
	random := rand.Float64()
	err := cs.writer.UpdateMetrics(&models.Metrics{
		ID:    randomValueMetric,
		MType: models.Gauge,
		Value: &random,
	})
	if err != nil {
		return fmt.Errorf("update %s metric error: %w", randomValueMetric, err)
	}
	return nil
}

func (cs *MetricsCollectService) updatePollCount() error {
	var pollCount int64 = 1
	err := cs.writer.UpdateMetrics(&models.Metrics{
		ID:    pollCountMetric,
		MType: models.Counter,
		Delta: &pollCount,
	})
	if err != nil {
		return fmt.Errorf("update %s metric error: %w", pollCountMetric, err)
	}
	return nil
}

func (cs *MetricsCollectService) UpdateAllMetrics() error {
	if err := cs.updateCollectMetrics(); err != nil {
		return err
	}

	if err := cs.updateRandomValue(); err != nil {
		return err
	}

	if err := cs.updatePollCount(); err != nil {
		return err
	}

	return nil
}

func (cs *MetricsCollectService) ResetPollCount() error {
	err := cs.writer.ResetMetricValue(&models.Metrics{
		ID:    pollCountMetric,
		MType: models.Counter,
	})

	if err != nil {
		return fmt.Errorf("reset %s metric error: %w", pollCountMetric, err)
	}

	return nil
}
