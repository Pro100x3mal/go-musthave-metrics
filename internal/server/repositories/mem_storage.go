package repositories

import (
	"errors"
	"sync"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
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

func (m *MemStorage) UpdateGauge(metric *models.Metrics) error {
	if metric.Value == nil {
		return errors.New("nil gauge value")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[metric.ID] = *metric.Value

	return nil
}

func (m *MemStorage) UpdateCounter(metric *models.Metrics) error {
	if metric.Delta == nil {
		return errors.New("nil counter delta")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[metric.ID] += *metric.Delta

	return nil
}

func (m *MemStorage) GetGauge(id string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, exist := m.gauges[id]
	if !exist {
		return 0, models.ErrMetricNotFound
	}
	return v, nil
}

func (m *MemStorage) GetCounter(id string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, exist := m.counters[id]
	if !exist {
		return 0, models.ErrMetricNotFound
	}
	return v, nil
}

func (m *MemStorage) GetAllGauges() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges
}

func (m *MemStorage) GetAllCounters() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters
}
