package repositories

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

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

func (db *DB) UpdateMetrics(metrics []models.Metrics) error {
	if metrics == nil {
		return errors.New("no metrics provided: slice is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, "insert_gauges", `
		INSERT INTO gauges (id, value)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET value = $2
	`)

	_, err = tx.Prepare(ctx, "insert_counters", `
		INSERT INTO counters (id, delta)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET delta = counters.delta + $2
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return errors.New("nil gauge value")
			}

			_, err = tx.Exec(ctx, "insert_gauges", metric.ID, *metric.Value)
			if err != nil {
				return fmt.Errorf("failed to insert/update %s metric '%s': %w", metric.MType, metric.ID, err)
			}
		case models.Counter:
			if metric.Delta == nil {
				return errors.New("nil counter delta")
			}
			_, err = tx.Exec(ctx, "insert_counters", metric.ID, *metric.Delta)
			if err != nil {
				return fmt.Errorf("failed to insert/update %s metric '%s': %w", metric.MType, metric.ID, err)
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (db *DB) UpdateGauge(metric *models.Metrics) error {
	if metric.Value == nil {
		return errors.New("nil gauge value")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO gauges (id, value)
		VALUES ( $1, $2 )
		ON CONFLICT (id) DO UPDATE
		SET value = $2
    `
	_, err := db.pool.Exec(ctx, query, metric.ID, *metric.Value)
	if err != nil {
		return fmt.Errorf("database error: failed to insert gauge metric: %w", err)
	}

	return nil
}

func (db *DB) UpdateCounter(metric *models.Metrics) error {
	if metric.Delta == nil {
		return errors.New("nil counter delta")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO counters (id, delta)
		VALUES ( $1, $2 )
		ON CONFLICT (id) DO UPDATE
		SET delta = counters.delta + $2
    `

	_, err := db.pool.Exec(ctx, query, metric.ID, *metric.Delta)
	if err != nil {
		return fmt.Errorf("database error: failed to insert counter metric: %w", err)
	}

	return nil
}

func (db *DB) GetGauge(id string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT value FROM gauges WHERE id = $1
	`

	row := db.pool.QueryRow(ctx, query, id)
	var v float64
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrMetricNotFound
		}
		return 0, fmt.Errorf("database error: failed to get gauge metric: %w", err)
	}
	return v, nil
}

func (db *DB) GetCounter(id string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT delta FROM counters WHERE id = $1
	`

	row := db.pool.QueryRow(ctx, query, id)
	var v int64
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrMetricNotFound
		}
		return 0, fmt.Errorf("database error: failed to get counter metric: %w", err)
	}
	return v, nil
}

func (db *DB) GetAllGauges() (map[string]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, value FROM gauges
	`
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query to get all gauge metrics: %w", err)
	}
	defer rows.Close()

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

	if len(m) == 0 {
		return nil, models.ErrMetricNotFound
	}

	return m, nil
}

func (db *DB) GetAllCounters() (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, delta FROM counters
	`
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query to get all counter metrics: %w", err)
	}
	defer rows.Close()

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

	if len(m) == 0 {
		return nil, models.ErrMetricNotFound
	}

	return m, nil
}
