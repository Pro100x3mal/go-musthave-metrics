package repositories

import "github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"

type RepositoryReader interface {
	GetGauge(mName string) (float64, error)
	GetCounter(mName string) (int64, error)
	GetAllGauges() (map[string]float64, error)
	GetAllCounters() (map[string]int64, error)
}

type RepositoryWriter interface {
	UpdateGauge(metric *models.Metrics) error
	UpdateCounter(metric *models.Metrics) error
	UpdateMetrics(metrics []models.Metrics) error
}

type Repository interface {
	RepositoryReader
	RepositoryWriter
}
