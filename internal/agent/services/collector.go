package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type RepositoryWriter interface {
	UpdateMetrics(metric *models.Metrics) error
	ResetMetricValue(metric *models.Metrics) error
}

type MetricsCollectService struct {
	writer     RepositoryWriter
	rntMetrics *runtimeMetricsProvider
	memMetrics *sysMemMetricsProvider
	cpuMetrics *sysCPUMetricsProvider
}

func NewMetricsCollectService(writer RepositoryWriter) *MetricsCollectService {
	return &MetricsCollectService{
		writer:     writer,
		rntMetrics: newRuntimeMetricsProvider(),
		memMetrics: newSysMemMetricsProvider(),
		cpuMetrics: newSysCPUMetricsProvider(),
	}
}

const (
	pollCountMetric   = "PollCount"
	randomValueMetric = "RandomValue"
)

type runtimeMetricsProvider struct {
	stats          *runtime.MemStats
	runtimeMetrics map[string]func(m *runtime.MemStats) float64
}

func newRuntimeMetricsProvider() *runtimeMetricsProvider {
	return &runtimeMetricsProvider{
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

func (cs *MetricsCollectService) updateRuntimeMetrics(ctx context.Context) error {
	runtime.ReadMemStats(cs.rntMetrics.stats)

	for name, fn := range cs.rntMetrics.runtimeMetrics {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		val := fn(cs.rntMetrics.stats)
		if err := cs.writer.UpdateMetrics(&models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &val,
		}); err != nil {
			return fmt.Errorf("update %s metric error: %w", name, err)
		}
	}

	return nil
}

func (cs *MetricsCollectService) updateRandomValue(ctx context.Context) error {
	random := rand.Float64()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := cs.writer.UpdateMetrics(&models.Metrics{
		ID:    randomValueMetric,
		MType: models.Gauge,
		Value: &random,
	}); err != nil {
		return fmt.Errorf("update %s metric error: %w", randomValueMetric, err)
	}

	return nil
}

func (cs *MetricsCollectService) updatePollCount(ctx context.Context) error {
	var pollCount int64 = 1

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := cs.writer.UpdateMetrics(&models.Metrics{
		ID:    pollCountMetric,
		MType: models.Counter,
		Delta: &pollCount,
	}); err != nil {
		return fmt.Errorf("update %s metric error: %w", pollCountMetric, err)
	}
	return nil
}

type sysMemMetricsProvider struct {
	systemMemoryMetrics map[string]func(*mem.VirtualMemoryStat) float64
}

func newSysMemMetricsProvider() *sysMemMetricsProvider {
	return &sysMemMetricsProvider{
		systemMemoryMetrics: map[string]func(vm *mem.VirtualMemoryStat) float64{
			"TotalMemory": func(vm *mem.VirtualMemoryStat) float64 { return float64(vm.Total) },
			"FreeMemory":  func(vm *mem.VirtualMemoryStat) float64 { return float64(vm.Free) },
		},
	}
}

func (cs *MetricsCollectService) updateSysMemoryMetrics(ctx context.Context) error {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("failed to get virtual memory statistics: %w", err)
	}

	for name, fn := range cs.memMetrics.systemMemoryMetrics {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		val := fn(vm)
		if err := cs.writer.UpdateMetrics(&models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &val,
		}); err != nil {
			return fmt.Errorf("update %s metric error: %w", name, err)
		}
	}

	return nil
}

type sysCPUMetricsProvider struct {
	cpuCount int
}

func newSysCPUMetricsProvider() *sysCPUMetricsProvider {
	count, err := cpu.Counts(true)
	if err != nil {
		count = runtime.NumCPU()
	}

	return &sysCPUMetricsProvider{
		cpuCount: count,
	}
}

func (cs *MetricsCollectService) updateCPUMetrics(ctx context.Context) error {
	percentages, err := cpu.Percent(time.Second, true)
	if err != nil {
		return fmt.Errorf("failed to get CPU utilization: %w", err)
	}

	expectedCount := cs.cpuMetrics.cpuCount
	if len(percentages) != expectedCount {
		return fmt.Errorf("expected %d CPU metrics, got %d", expectedCount, len(percentages))
	}

	for i, percentage := range percentages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		metricName := fmt.Sprintf("CPUutilization%d", i)
		val := percentage

		if err := cs.writer.UpdateMetrics(&models.Metrics{
			ID:    metricName,
			MType: models.Gauge,
			Value: &val,
		}); err != nil {
			return fmt.Errorf("update %s metric error: %w", metricName, err)
		}
	}

	return nil
}

func (cs *MetricsCollectService) UpdateAllMetrics(ctx context.Context) error {
	if err := cs.updateRuntimeMetrics(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to update metrics: %w", err)
	}

	if err := cs.updateSysMemoryMetrics(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to update metrics: %w", err)
	}

	if err := cs.updateCPUMetrics(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to update CPU metrics: %w", err)
	}

	if err := cs.updateRandomValue(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to update metrics: %w", err)
	}

	if err := cs.updatePollCount(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to update metrics: %w", err)
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
