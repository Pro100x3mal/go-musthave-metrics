package service

import (
	"math/rand"
	"runtime"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/model"
)

type MetricsRepository interface {
	UpdateMetrics(metric *model.Metrics) error
	GetAllMetrics() []*model.Metrics
}

type MetricsService struct {
	repo      MetricsRepository
	pollCount int64
}

func NewMetricsService(repo MetricsRepository) *MetricsService {
	return &MetricsService{
		repo: repo,
	}
}

type getMetrics struct {
	name string
	get  func(m *runtime.MemStats) float64
}

var runtimeMetrics = []getMetrics{
	{name: "MetricAlloc", get: func(m *runtime.MemStats) float64 { return float64(m.Alloc) }},
	{name: "MetricBuckHashSys", get: func(m *runtime.MemStats) float64 { return float64(m.BuckHashSys) }},
	{name: "MetricFrees", get: func(m *runtime.MemStats) float64 { return float64(m.Frees) }},
	{name: "MetricGCCPUFraction", get: func(m *runtime.MemStats) float64 { return m.GCCPUFraction }},
	{name: "MetricGCSys", get: func(m *runtime.MemStats) float64 { return float64(m.GCSys) }},
	{name: "MetricHeapAlloc", get: func(m *runtime.MemStats) float64 { return float64(m.HeapAlloc) }},
	{name: "MetricHeapIdle", get: func(m *runtime.MemStats) float64 { return float64(m.HeapIdle) }},
	{name: "MetricHeapInuse", get: func(m *runtime.MemStats) float64 { return float64(m.HeapInuse) }},
	{name: "MetricHeapObjects", get: func(m *runtime.MemStats) float64 { return float64(m.HeapObjects) }},
	{name: "MetricHeapReleased", get: func(m *runtime.MemStats) float64 { return float64(m.HeapReleased) }},
	{name: "MetricHeapSys", get: func(m *runtime.MemStats) float64 { return float64(m.HeapSys) }},
	{name: "MetricLastGC", get: func(m *runtime.MemStats) float64 { return float64(m.LastGC) }},
	{name: "MetricLookups", get: func(m *runtime.MemStats) float64 { return float64(m.Lookups) }},
	{name: "MetricMCacheInuse", get: func(m *runtime.MemStats) float64 { return float64(m.MCacheInuse) }},
	{name: "MetricMCacheSys", get: func(m *runtime.MemStats) float64 { return float64(m.MCacheSys) }},
	{name: "MetricMSpanInuse", get: func(m *runtime.MemStats) float64 { return float64(m.MSpanInuse) }},
	{name: "MetricMSpanSys", get: func(m *runtime.MemStats) float64 { return float64(m.MSpanSys) }},
	{name: "MetricMallocs", get: func(m *runtime.MemStats) float64 { return float64(m.Mallocs) }},
	{name: "MetricNextGC", get: func(m *runtime.MemStats) float64 { return float64(m.NextGC) }},
	{name: "MetricNumForcedGC", get: func(m *runtime.MemStats) float64 { return float64(m.NumForcedGC) }},
	{name: "MetricNumGC", get: func(m *runtime.MemStats) float64 { return float64(m.NumGC) }},
	{name: "MetricOtherSys", get: func(m *runtime.MemStats) float64 { return float64(m.OtherSys) }},
	{name: "MetricPauseTotalNs", get: func(m *runtime.MemStats) float64 { return float64(m.PauseTotalNs) }},
	{name: "MetricStackInuse", get: func(m *runtime.MemStats) float64 { return float64(m.StackInuse) }},
	{name: "MetricStackSys", get: func(m *runtime.MemStats) float64 { return float64(m.StackSys) }},
	{name: "MetricSys", get: func(m *runtime.MemStats) float64 { return float64(m.Sys) }},
	{name: "MetricTotalAlloc", get: func(m *runtime.MemStats) float64 { return float64(m.TotalAlloc) }},
}

func (ms *MetricsService) CollectMetrics() error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	for _, getFunc := range runtimeMetrics {
		val := getFunc.get(&m)
		err := ms.repo.UpdateMetrics(&model.Metrics{
			ID:    getFunc.name,
			MType: model.Gauge,
			Value: &val,
		})
		if err != nil {
			return err
		}
	}

	err := ms.updateRandomValue()
	if err != nil {
		return err
	}

	err = ms.incrementPollCount()
	if err != nil {
		return err
	}

	return nil
}

func (ms *MetricsService) updateRandomValue() error {
	random := rand.Float64()
	err := ms.repo.UpdateMetrics(&model.Metrics{
		ID:    "RandomValue",
		MType: model.Gauge,
		Value: &random,
	})
	if err != nil {
		return err
	}
	return nil
}

func (ms *MetricsService) incrementPollCount() error {
	ms.pollCount++
	val := ms.pollCount
	err := ms.repo.UpdateMetrics(&model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &val,
	})
	if err != nil {
		return err
	}
	return nil
}
