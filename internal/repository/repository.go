package repository

import "sync"

type MetricsStorage interface {
	UpdateGauge(id string, value float64)
	UpdateCounter(id string, delta int64)
}

type MemStorage struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MemStorage) UpdateGauge(id string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[id] = value
}

func (m *MemStorage) UpdateCounter(id string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[id] += delta
}
