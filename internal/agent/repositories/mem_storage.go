package repositories

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
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

func (m *MemStorage) UpdateMetrics(metric *models.Metrics) error {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return errors.New("nil gauge value")
		}
		m.mu.Lock()
		m.gauges[metric.ID] = *metric.Value
		m.mu.Unlock()
	case models.Counter:
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

func (m *MemStorage) ResetMetricValue(metric *models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch metric.MType {
	case models.Gauge:
		if _, ok := m.gauges[metric.ID]; !ok {
			return errors.New("gauge metric not found")
		}
		m.gauges[metric.ID] = 0
	case models.Counter:
		if _, ok := m.counters[metric.ID]; !ok {
			return errors.New("counter metric not found")
		}
		m.counters[metric.ID] = 0
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.MType)
	}
	return nil
}

func (m *MemStorage) GetAllMetrics() []*models.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Metrics

	for id, value := range m.gauges {
		v := value
		result = append(result, &models.Metrics{
			ID:    id,
			MType: models.Gauge,
			Value: &v,
		})
	}

	for id, delta := range m.counters {
		d := delta
		result = append(result, &models.Metrics{
			ID:    id,
			MType: models.Counter,
			Delta: &d,
		})
	}

	return result
}
