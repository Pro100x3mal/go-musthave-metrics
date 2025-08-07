package repositories

import (
	"context"
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

func (m *MemStorage) UpdateGauge(_ context.Context, metric *models.Metrics) error {
	if metric.Value == nil {
		return errors.New("nil gauge value")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[metric.ID] = *metric.Value

	return nil
}

func (m *MemStorage) UpdateCounter(_ context.Context, metric *models.Metrics) error {
	if metric.Delta == nil {
		return errors.New("nil counter delta")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[metric.ID] += *metric.Delta

	return nil
}

func (m *MemStorage) UpdateMetrics(_ context.Context, metrics []models.Metrics) error {
	if metrics == nil {
		return errors.New("no metrics provided: slice is nil")
	}
	for _, metric := range metrics {
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
		}
	}

	return nil
}

func (m *MemStorage) GetGauge(_ context.Context, id string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, exist := m.gauges[id]
	if !exist {
		return 0, models.ErrMetricNotFound
	}
	return v, nil
}

func (m *MemStorage) GetCounter(_ context.Context, id string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, exist := m.counters[id]
	if !exist {
		return 0, models.ErrMetricNotFound
	}
	return v, nil
}

func (m *MemStorage) GetAllGauges(_ context.Context) (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.gauges == nil {
		return nil, models.ErrMetricNotFound
	}
	return m.gauges, nil
}

func (m *MemStorage) GetAllCounters(_ context.Context) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.counters == nil {
		return nil, models.ErrMetricNotFound
	}
	return m.counters, nil
}
