package repository

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/model"
)

type MemStorage struct {
	mu       *sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		mu:       &sync.Mutex{},
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
		defer m.mu.Unlock()
		m.gauges[metric.ID] = *metric.Value
	case model.Counter:
		if metric.Delta == nil {
			return errors.New("nil counter delta")
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		m.counters[metric.ID] += *metric.Delta
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.MType)
	}

	return nil
}

func (m *MemStorage) GetAllMetrics() []*model.Metrics {
	m.mu.Lock()
	defer m.mu.Unlock()

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
