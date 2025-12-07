package transport

import (
	"context"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
)

type MetricsProvider interface {
	GetAllMetrics() []*models.Metrics
}

type Sender interface {
	SendMetrics(ctx context.Context) error
}
