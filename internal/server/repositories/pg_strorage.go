package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, cfg *configs.ServerConfig) (*DB, error) {
	pool, err := initPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	m, err := migrate.New("file://internal/server/repositories/db/migrations", cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, err
	}

	return &DB{
		pool: pool,
	}, nil
}

func initPool(ctx context.Context, cfg *configs.ServerConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
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

func (db *DB) UpdateGauge(metric *models.Metrics) error {
	if metric.Value == nil {
		return errors.New("nil gauge value")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO metrics (id, mtype, value)
		VALUES ( $1, $2, $3 )
		ON CONFLICT (id) DO UPDATE
		SET mtype = $2, value = $3
    `
	_, err := db.pool.Exec(ctx, query, metric.ID, metric.MType, *metric.Value)
	if err != nil {
		return fmt.Errorf("database error: failed to insert metric: %w", err)
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
		INSERT INTO metrics (id, mtype, delta)
		VALUES ( $1, $2, $3 )
		ON CONFLICT (id) DO UPDATE
		SET mtype = $2, delta = $3
    `

	_, err := db.pool.Exec(ctx, query, metric.ID, metric.MType, *metric.Delta)
	if err != nil {
		return fmt.Errorf("database error: failed to insert metric: %w", err)
	}

	return nil
}

func (db *DB) GetGauge(id string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT value FROM metrics WHERE id = $1
	`

	row := db.pool.QueryRow(ctx, query, id)
	var v float64
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrMetricNotFound
		}
		return 0, fmt.Errorf("database error: failed to get metric: %w", err)
	}
	return v, nil
}

func (db *DB) GetCounter(id string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT delta FROM metrics WHERE id = $1
	`

	row := db.pool.QueryRow(ctx, query, id)
	var v int64
	if err := row.Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrMetricNotFound
		}
		return 0, fmt.Errorf("database error: failed to get metric: %w", err)
	}
	return v, nil
}

func (db *DB) GetAllGauges() (map[string]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, value FROM metrics
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
			return nil, fmt.Errorf("failed to scan gauge metric values from database: %w", err)
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
		SELECT id, delta FROM metrics
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
			return nil, fmt.Errorf("failed to scan counter metric values from database: %w", err)
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
