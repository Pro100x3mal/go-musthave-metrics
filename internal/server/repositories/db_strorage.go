package repositories

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var retryIntervals = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, cfg *configs.ServerConfig, logger *zap.Logger) (*DB, error) {
	dbLogger := logger.With(zap.String("component", "db"))
	dbLogger.Info("initializing database storage", zap.String("dsn", cfg.DatabaseDSN))

	dbLogger.Info("running database migrations")
	err := runMigrations(cfg)
	if err != nil {
		dbLogger.Error("failed to run database migrations", zap.Error(err))
		return nil, err
	}
	dbLogger.Info("database migrations completed")

	dbLogger.Info("connecting to database")
	pool, err := initPool(ctx, cfg)
	if err != nil {
		dbLogger.Error("failed to connect to database", zap.Error(err))
		return nil, err
	}
	dbLogger.Info("connected to database")

	dbLogger.Info("database storage initialized successfully")

	return &DB{
		pool: pool,
	}, nil
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func runMigrations(cfg *configs.ServerConfig) error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to return an iofs driver: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, cfg.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("failed to initialize a migration instance: %w", err)
	}
	defer m.Close()

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations to the DB: %w", err)
	}
	return nil
}

func initPool(ctx context.Context, cfg *configs.ServerConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN %s: %w", cfg.DatabaseDSN, err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	err := db.pool.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
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

func (db *DB) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	if metrics == nil {
		return errors.New("no metrics provided: slice is nil")
	}
	gaugeMap := make(map[string]float64)
	counterMap := make(map[string]int64)

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				return errors.New("nil gauge value")
			}
			gaugeMap[m.ID] = *m.Value
		case models.Counter:
			if m.Delta == nil {
				return errors.New("nil counter delta")
			}
			counterMap[m.ID] += *m.Delta
		}
	}
	var gauges, counters []models.Metrics
	for id, v := range gaugeMap {
		value := v
		gauges = append(gauges, models.Metrics{
			ID:    id,
			Value: &value,
			MType: models.Gauge,
		})
	}
	for id, d := range counterMap {
		delta := d
		counters = append(counters, models.Metrics{
			ID:    id,
			Delta: &delta,
			MType: models.Counter,
		})
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	maxRetries := len(retryIntervals) + 1

	if len(gauges) > 0 {
		var query strings.Builder
		query.WriteString("INSERT INTO gauges (id, value) VALUES ")
		var args []any
		for i, m := range gauges {
			n := i*2 + 1
			query.WriteString(fmt.Sprintf("($%d, $%d)", n, n+1))
			if i < len(gauges)-1 {
				query.WriteString(", ")
			}
			args = append(args, m.ID, *m.Value)
		}
		query.WriteString(" ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value")

		var lastErr error
		for attempt := 0; attempt < maxRetries; attempt++ {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			_, err = tx.Exec(ctx, query.String(), args...)
			cancel()

			if err == nil {
				lastErr = nil
				break
			}
			lastErr = err
			if !isRetryableDBError(err) {
				return fmt.Errorf("failed to update gauge batch: %w", err)
			}
			if attempt < maxRetries-1 {
				time.Sleep(retryIntervals[attempt])
			}
		}
		if lastErr != nil {
			return fmt.Errorf("failed to update gauge batch after retries: %w", lastErr)
		}
	}

	if len(counters) > 0 {
		var query strings.Builder
		query.WriteString("INSERT INTO counters (id, delta) VALUES ")
		var args []any
		for i, m := range counters {
			n := i*2 + 1
			query.WriteString(fmt.Sprintf("($%d, $%d)", n, n+1))
			if i < len(counters)-1 {
				query.WriteString(", ")
			}
			args = append(args, m.ID, *m.Delta)
		}
		query.WriteString(" ON CONFLICT (id) DO UPDATE SET delta = counters.delta + EXCLUDED.delta")

		var lastErr error
		for attempt := 0; attempt < maxRetries; attempt++ {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			_, err = tx.Exec(ctx, query.String(), args...)
			cancel()

			if err == nil {
				lastErr = nil
				break
			}
			lastErr = err
			if !isRetryableDBError(err) {
				return fmt.Errorf("failed to update counter batch: %w", err)
			}
			if attempt < maxRetries-1 {
				time.Sleep(retryIntervals[attempt])
			}
		}
		if lastErr != nil {
			return fmt.Errorf("failed to update counter batch after retries: %w", lastErr)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (db *DB) UpdateGauge(ctx context.Context, metric *models.Metrics) error {
	if metric.Value == nil {
		return errors.New("nil gauge value")
	}

	query := `
		INSERT INTO gauges (id, value)
		VALUES ( $1, $2 )
		ON CONFLICT (id) DO UPDATE
		SET value = $2
    `

	maxRetries := len(retryIntervals) + 1
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctxRetry, cancelRetry := context.WithTimeout(ctx, 3*time.Second)

		_, err := db.pool.Exec(ctxRetry, query, metric.ID, *metric.Value)
		cancelRetry()

		if err == nil {
			return nil
		}

		lastErr = err
		if !isRetryableDBError(err) {
			return fmt.Errorf("database error: failed to insert gauge metric: %w", err)
		}

		if attempt < maxRetries-1 {
			time.Sleep(retryIntervals[attempt])
		}
	}
	return fmt.Errorf("database error: failed to insert gauge metric after retries: %w", lastErr)
}

func (db *DB) UpdateCounter(ctx context.Context, metric *models.Metrics) error {
	if metric.Delta == nil {
		return errors.New("nil counter delta")
	}

	query := `
		INSERT INTO counters (id, delta)
		VALUES ( $1, $2 )
		ON CONFLICT (id) DO UPDATE
		SET delta = counters.delta + $2
    `

	maxRetries := len(retryIntervals) + 1
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctxRetry, cancelRetry := context.WithTimeout(ctx, 3*time.Second)

		_, err := db.pool.Exec(ctxRetry, query, metric.ID, *metric.Delta)
		cancelRetry()

		if err == nil {
			return nil
		}

		lastErr = err
		if !isRetryableDBError(err) {
			return fmt.Errorf("database error: failed to insert counter metric: %w", err)
		}

		if attempt < maxRetries-1 {
			time.Sleep(retryIntervals[attempt])
		}
	}
	return fmt.Errorf("database error: failed to insert counter metric after retries: %w", lastErr)
}

func (db *DB) GetGauge(ctx context.Context, id string) (float64, error) {
	query := `
		SELECT value FROM gauges WHERE id = $1
	`

	maxRetries := len(retryIntervals) + 1
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctxRetry, cancelRetry := context.WithTimeout(ctx, 3*time.Second)

		row := db.pool.QueryRow(ctxRetry, query, id)
		var v float64
		err := row.Scan(&v)
		cancelRetry()

		if err == nil {
			return v, nil
		}

		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrMetricNotFound
		}

		lastErr = err
		if !isRetryableDBError(err) {
			return 0, fmt.Errorf("database error: failed to get gauge metric: %w", err)
		}

		if attempt < maxRetries-1 {
			time.Sleep(retryIntervals[attempt])
		}
	}
	return 0, fmt.Errorf("database error: failed to get gauge metric after retries: %w", lastErr)
}

func (db *DB) GetCounter(ctx context.Context, id string) (int64, error) {
	query := `
		SELECT delta FROM counters WHERE id = $1
	`

	maxRetries := len(retryIntervals) + 1
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctxRetry, cancelRetry := context.WithTimeout(ctx, 3*time.Second)

		row := db.pool.QueryRow(ctxRetry, query, id)
		var v int64
		err := row.Scan(&v)
		cancelRetry()

		if err == nil {
			return v, nil
		}

		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrMetricNotFound
		}

		lastErr = err
		if !isRetryableDBError(err) {
			return 0, fmt.Errorf("database error: failed to get counter metric: %w", err)
		}

		if attempt < maxRetries-1 {
			time.Sleep(retryIntervals[attempt])
		}
	}
	return 0, fmt.Errorf("database error: failed to get counter metric after retries: %w", lastErr)
}

func (db *DB) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	query := `
		SELECT id, value FROM gauges
	`

	maxRetries := len(retryIntervals) + 1
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctxRetry, cancelRetry := context.WithTimeout(ctx, 3*time.Second)

		rows, err := db.pool.Query(ctxRetry, query)
		cancelRetry()

		if err == nil {
			m := make(map[string]float64)
			for rows.Next() {
				var id string
				var value float64

				if err = rows.Scan(&id, &value); err != nil {
					return nil, fmt.Errorf("failed to scan gauge metrics values from database: %w", err)
				}
				m[id] = value
			}

			if err = rows.Err(); err != nil {
				return nil, fmt.Errorf("error occurred while iterating over gauge metrics: %w", err)
			}

			rows.Close()

			if len(m) == 0 {
				return nil, models.ErrMetricNotFound
			}

			return m, nil
		}

		lastErr = err
		if !isRetryableDBError(err) {
			return nil, fmt.Errorf("failed to execute query to get all gauge metrics: %w", err)
		}

		if attempt < maxRetries-1 {
			time.Sleep(retryIntervals[attempt])
		}
	}
	return nil, fmt.Errorf("failed to get all gauge metrics after retries: %w", lastErr)
}

func (db *DB) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT id, delta FROM counters
	`

	maxRetries := len(retryIntervals) + 1
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctxRetry, cancelRetry := context.WithTimeout(ctx, 3*time.Second)

		rows, err := db.pool.Query(ctxRetry, query)
		cancelRetry()

		if err == nil {
			m := make(map[string]int64)
			for rows.Next() {
				var id string
				var delta int64

				if err = rows.Scan(&id, &delta); err != nil {
					return nil, fmt.Errorf("failed to scan counter metrics values from database: %w", err)
				}
				m[id] = delta
			}

			if err = rows.Err(); err != nil {
				return nil, fmt.Errorf("error occurred while iterating over counter metrics: %w", err)
			}

			rows.Close()

			if len(m) == 0 {
				return nil, models.ErrMetricNotFound
			}

			return m, nil
		}

		lastErr = err
		if !isRetryableDBError(err) {
			return nil, fmt.Errorf("failed to execute query to get all counter metrics: %w", err)
		}

		if attempt < maxRetries-1 {
			time.Sleep(retryIntervals[attempt])
		}
	}
	return nil, fmt.Errorf("failed to get all counter metrics after retries: %w", lastErr)
}
