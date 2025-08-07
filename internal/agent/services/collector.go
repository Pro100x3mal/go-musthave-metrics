package services

import (
	"context"
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
			"Alloc":         func(m *runtime.MemStats) float64 { return float64(m.Alloc) },
			"BuckHashSys":   func(m *runtime.MemStats) float64 { return float64(m.BuckHashSys) },
			"Frees":         func(m *runtime.MemStats) float64 { return float64(m.Frees) },
			"GCCPUFraction": func(m *runtime.MemStats) float64 { return m.GCCPUFraction },
			"GCSys":         func(m *runtime.MemStats) float64 { return float64(m.GCSys) },
			"HeapAlloc":     func(m *runtime.MemStats) float64 { return float64(m.HeapAlloc) },
			"HeapIdle":      func(m *runtime.MemStats) float64 { return float64(m.HeapIdle) },
			"HeapInuse":     func(m *runtime.MemStats) float64 { return float64(m.HeapInuse) },
			"HeapObjects":   func(m *runtime.MemStats) float64 { return float64(m.HeapObjects) },
			"HeapReleased":  func(m *runtime.MemStats) float64 { return float64(m.HeapReleased) },
			"HeapSys":       func(m *runtime.MemStats) float64 { return float64(m.HeapSys) },
			"LastGC":        func(m *runtime.MemStats) float64 { return float64(m.LastGC) },
			"Lookups":       func(m *runtime.MemStats) float64 { return float64(m.Lookups) },
			"MCacheInuse":   func(m *runtime.MemStats) float64 { return float64(m.MCacheInuse) },
			"MCacheSys":     func(m *runtime.MemStats) float64 { return float64(m.MCacheSys) },
			"MSpanInuse":    func(m *runtime.MemStats) float64 { return float64(m.MSpanInuse) },
			"MSpanSys":      func(m *runtime.MemStats) float64 { return float64(m.MSpanSys) },
			"Mallocs":       func(m *runtime.MemStats) float64 { return float64(m.Mallocs) },
			"NextGC":        func(m *runtime.MemStats) float64 { return float64(m.NextGC) },
			"NumForcedGC":   func(m *runtime.MemStats) float64 { return float64(m.NumForcedGC) },
			"NumGC":         func(m *runtime.MemStats) float64 { return float64(m.NumGC) },
			"OtherSys":      func(m *runtime.MemStats) float64 { return float64(m.OtherSys) },
			"PauseTotalNs":  func(m *runtime.MemStats) float64 { return float64(m.PauseTotalNs) },
			"StackInuse":    func(m *runtime.MemStats) float64 { return float64(m.StackInuse) },
			"StackSys":      func(m *runtime.MemStats) float64 { return float64(m.StackSys) },
			"Sys":           func(m *runtime.MemStats) float64 { return float64(m.Sys) },
			"TotalAlloc":    func(m *runtime.MemStats) float64 { return float64(m.TotalAlloc) },
		},
	}
}

func (cs *MetricsCollectService) updateCollectMetrics(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

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

func (cs *MetricsCollectService) updateRandomValue(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

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

func (cs *MetricsCollectService) updatePollCount(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

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

func (cs *MetricsCollectService) UpdateAllMetrics(ctx context.Context) error {
	if err := cs.updateCollectMetrics(ctx); err != nil {
		return err
	}

	if err := cs.updateRandomValue(ctx); err != nil {
		return err
	}

	if err := cs.updatePollCount(ctx); err != nil {
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
