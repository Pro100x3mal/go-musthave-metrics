package repository

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/model"
)

type MemStorage struct {
	mu       *sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		mu:       &sync.RWMutex{},
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MemStorage) UpdateMetrics(metric *model.Metrics) error {
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return errors.New("nil gauge value")
		}
		m.mu.Lock()
		m.gauges[metric.ID] = *metric.Value
		m.mu.Unlock()
	case model.Counter:
		if metric.Delta == nil {
			return errors.New("nil counter delta")
		}
		m.mu.Lock()
		m.counters[metric.ID] += *metric.Delta
		m.mu.Unlock()
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.MType)
	}

	return nil
}

func (m *MemStorage) ResetMetricValue(metric *model.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch metric.MType {
	case model.Gauge:
		if _, ok := m.gauges[metric.ID]; !ok {
			return errors.New("gauge metric not found")
		}
		m.gauges[metric.ID] = 0
	case model.Counter:
		if _, ok := m.counters[metric.ID]; !ok {
			return errors.New("counter metric not found")
		}
		m.counters[metric.ID] = 0
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.MType)
	}
	return nil
}

func (m *MemStorage) GetAllMetrics() []*model.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*model.Metrics

	for id, value := range m.gauges {
		v := value
		result = append(result, &model.Metrics{
			ID:    id,
			MType: model.Gauge,
			Value: &v,
		})
	}

	for id, delta := range m.counters {
		d := delta
		result = append(result, &model.Metrics{
			ID:    id,
			MType: model.Counter,
			Delta: &d,
		})
	}

	return result
}
