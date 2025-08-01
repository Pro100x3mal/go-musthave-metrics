package repositories

import (
	"context"
	"fmt"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
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

	return nil
}

func (db *DB) UpdateCounter(metric *models.Metrics) error {

	return nil
}

func (db *DB) GetGauge(id string) (float64, error) {

	return 0, nil
}

func (db *DB) GetCounter(id string) (int64, error) {

	return 0, nil
}

func (db *DB) GetAllGauges() map[string]float64 {
	return nil
}

func (db *DB) GetAllCounters() map[string]int64 {
	return nil
}
