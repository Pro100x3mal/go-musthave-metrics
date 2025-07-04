package service

import (
	"math/rand"
	"runtime"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/model"
)

type RepositoryReader interface {
	GetAllMetrics() []*model.Metrics
}

type RepositoryWriter interface {
	UpdateMetrics(metric *model.Metrics) error
}

type MetricsCollectService struct {
	writer    RepositoryWriter
	pollCount int64
}

type MetricsQueryService struct {
	reader RepositoryReader
}

func NewMetricsQueryService(reader RepositoryReader) *MetricsQueryService {
	return &MetricsQueryService{
		reader: reader,
	}
}

func NewMetricsCollectService(writer RepositoryWriter) *MetricsCollectService {
	return &MetricsCollectService{
		writer: writer,
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

func (cs *MetricsCollectService) CollectMetrics() error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	for _, getFunc := range runtimeMetrics {
		val := getFunc.get(&m)
		err := cs.writer.UpdateMetrics(&model.Metrics{
			ID:    getFunc.name,
			MType: model.Gauge,
			Value: &val,
		})
		if err != nil {
			return err
		}
	}

	err := cs.updateRandomValue()
	if err != nil {
		return err
	}

	err = cs.incrementPollCount()
	if err != nil {
		return err
	}

	return nil
}

func (cs *MetricsCollectService) updateRandomValue() error {
	random := rand.Float64()
	err := cs.writer.UpdateMetrics(&model.Metrics{
		ID:    "RandomValue",
		MType: model.Gauge,
		Value: &random,
	})
	if err != nil {
		return err
	}
	return nil
}

func (cs *MetricsCollectService) incrementPollCount() error {
	cs.pollCount++
	val := cs.pollCount
	err := cs.writer.UpdateMetrics(&model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &val,
	})
	if err != nil {
		return err
	}
	return nil
}

func (qs *MetricsQueryService) GetAllMetrics() []*model.Metrics {
	return qs.reader.GetAllMetrics()
}
