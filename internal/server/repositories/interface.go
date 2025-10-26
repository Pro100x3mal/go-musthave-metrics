package repositories

import (
	"context"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
)

// RepositoryReader provides read-only operations for metrics storage.
type RepositoryReader interface {
	GetGauge(ctx context.Context, mName string) (float64, error)
	GetCounter(ctx context.Context, mName string) (int64, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)
}

// RepositoryWriter provides write operations for metrics storage.
type RepositoryWriter interface {
	UpdateGauge(ctx context.Context, metric *models.Metrics) error
	UpdateCounter(ctx context.Context, metric *models.Metrics) error
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
}

// Repository combines read and write operations for metrics storage.
type Repository interface {
	RepositoryReader
	RepositoryWriter
}
