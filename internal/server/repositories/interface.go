package repositories

import (
	"context"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
)

type RepositoryReader interface {
	GetGauge(ctx context.Context, mName string) (float64, error)
	GetCounter(ctx context.Context, mName string) (int64, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)
}

type RepositoryWriter interface {
	UpdateGauge(ctx context.Context, metric *models.Metrics) error
	UpdateCounter(ctx context.Context, metric *models.Metrics) error
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
}

type Repository interface {
	RepositoryReader
	RepositoryWriter
}
