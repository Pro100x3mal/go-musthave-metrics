package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type RepoWithRetry struct {
	inner          repositories.Repository
	intervals      []time.Duration
	attemptTimeout time.Duration
}

func NewRepoWithRetry(inner repositories.Repository, intervals []time.Duration, attemptTimeout time.Duration) *RepoWithRetry {
	if len(intervals) == 0 {
		intervals = []time.Duration{
			1 * time.Second,
			3 * time.Second,
			5 * time.Second,
		}
	}

	if attemptTimeout <= 0 {
		attemptTimeout = 5 * time.Second
	}

	return &RepoWithRetry{
		inner:          inner,
		intervals:      intervals,
		attemptTimeout: attemptTimeout,
	}
}

func isRetryableDBError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		code := pgErr.Code
		switch code {
		case "08000", "08003", "08006", "08001", "08004":
			return true
		case "57014", "57P01", "57P02", "57P03":
			return true
		case "40001", "40P01":
			return true
		default:
			return false
		}
	}
	return false
}

func (r *RepoWithRetry) withRetry(ctx context.Context, fn func(context.Context) error) error {
	maxRetries := len(r.intervals) + 1
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		retryCtx, cancel := context.WithTimeout(ctx, r.attemptTimeout)

		err := fn(retryCtx)
		cancel()

		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		if errors.Is(err, models.ErrMetricNotFound) || errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		lastErr = err

		if i < maxRetries-1 && isRetryableDBError(err) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(r.intervals[i]):
				continue
			}
		}
		return err
	}
	return fmt.Errorf("operation failed after retries: %w", lastErr)
}

func (r *RepoWithRetry) UpdateGauge(ctx context.Context, metric *models.Metrics) error {
	return r.withRetry(ctx, func(retryCtx context.Context) error {
		return r.inner.UpdateGauge(retryCtx, metric)
	})
}

func (r *RepoWithRetry) UpdateCounter(ctx context.Context, metric *models.Metrics) error {
	return r.withRetry(ctx, func(retryCtx context.Context) error {
		return r.inner.UpdateCounter(retryCtx, metric)
	})
}

func (r *RepoWithRetry) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	return r.withRetry(ctx, func(retryCtx context.Context) error {
		return r.inner.UpdateMetrics(retryCtx, metrics)
	})
}

func (r *RepoWithRetry) GetGauge(ctx context.Context, id string) (float64, error) {
	var out float64
	err := r.withRetry(ctx, func(retryCtx context.Context) error {
		value, err := r.inner.GetGauge(retryCtx, id)
		if err != nil {
			return err
		}
		out = value
		return nil
	})
	return out, err
}

func (r *RepoWithRetry) GetCounter(ctx context.Context, id string) (int64, error) {
	var out int64
	err := r.withRetry(ctx, func(retryCtx context.Context) error {
		delta, err := r.inner.GetCounter(retryCtx, id)
		if err != nil {
			return err
		}
		out = delta
		return nil
	})
	return out, err
}

func (r *RepoWithRetry) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	var out map[string]float64
	err := r.withRetry(ctx, func(retryCtx context.Context) error {
		gauges, err := r.inner.GetAllGauges(retryCtx)
		if err != nil {
			return err
		}
		out = gauges
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *RepoWithRetry) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	var out map[string]int64
	err := r.withRetry(ctx, func(retryCtx context.Context) error {
		counters, err := r.inner.GetAllCounters(retryCtx)
		if err != nil {
			return err
		}
		out = counters
		return nil
	})

	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *RepoWithRetry) Ping(ctx context.Context) error {
	type dbPinger interface {
		Ping(ctx context.Context) error
	}
	p, ok := r.inner.(dbPinger)
	if !ok {
		return errors.New("pinging not supported by this repository")
	}

	return r.withRetry(ctx, func(retrytCtx context.Context) error {
		return p.Ping(retrytCtx)
	})
}
