package repositories

import (
	"context"
	"fmt"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
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
	return db.pool.Ping(ctx)
}
