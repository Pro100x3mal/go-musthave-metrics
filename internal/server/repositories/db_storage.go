package repositories

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
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

const defaultChunkSize = 100

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, cfg *configs.ServerConfig, logger *zap.Logger) (*DB, error) {
	logger.Info("initializing database storage", zap.String("dsn", cfg.DatabaseDSN))

	logger.Debug("running database migrations")
	err := runMigrations(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}
	logger.Debug("database migrations completed")

	logger.Debug("connecting to database")
	pool, err := initPool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	logger.Debug("connected to database")

	logger.Info("database storage initialized successfully")

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

	poolCfg.MaxConns = 50
	poolCfg.MinConns = 10
	poolCfg.MaxConnLifetime = 1 * time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = 1 * time.Minute

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

	gauges := make([]models.Metrics, 0, len(gaugeMap))
	for id, v := range gaugeMap {
		value := v
		gauges = append(gauges, models.Metrics{
			ID:    id,
			Value: &value,
			MType: models.Gauge,
		})
	}

	counters := make([]models.Metrics, 0, len(counterMap))
	for id, d := range counterMap {
		delta := d
		counters = append(counters, models.Metrics{
			ID:    id,
			Delta: &delta,
			MType: models.Counter,
		})
	}

	sort.Slice(gauges, func(i, j int) bool {
		return gauges[i].ID < gauges[j].ID
	})

	sort.Slice(counters, func(i, j int) bool {
		return counters[i].ID < counters[j].ID
	})

	gaugeChunks := splitMetricsIntoChunks(gauges, defaultChunkSize)
	counterChunks := splitMetricsIntoChunks(counters, defaultChunkSize)

	for i, chunk := range gaugeChunks {
		if err := db.updateMetricsChunk(ctx, chunk, nil); err != nil {
			return fmt.Errorf("failed to update gauge chunk %d/%d: %w", i+1, len(gaugeChunks), err)
		}
	}

	for i, chunk := range counterChunks {
		if err := db.updateMetricsChunk(ctx, nil, chunk); err != nil {
			return fmt.Errorf("failed to update counter chunk %d/%d: %w", i+1, len(counterChunks), err)
		}
	}

	return nil
}

func splitMetricsIntoChunks(items []models.Metrics, chunkSize int) [][]models.Metrics {
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	if len(items) == 0 {
		return nil
	}

	chunks := make([][]models.Metrics, 0, len(items)/chunkSize+1)
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

func (db *DB) updateMetricsChunk(ctx context.Context, gauges, counters []models.Metrics) error {
	if len(gauges) == 0 && len(counters) == 0 {
		return nil
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if len(gauges) > 0 {
		values := make([]string, 0, len(gauges))
		args := make([]any, 0, len(gauges)*2)
		for i, m := range gauges {
			base := i * 2
			params := fmt.Sprintf("($%d, $%d)", base+1, base+2)
			values = append(values, params)
			args = append(args, m.ID, *m.Value)
		}

		query := `INSERT INTO gauges (id, value) VALUES ` + strings.Join(values, ",") + ` ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value`

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return err
			}
			return fmt.Errorf("failed to insert gauge batch: %w", err)
		}
	}

	if len(counters) > 0 {
		values := make([]string, 0, len(counters))
		args := make([]any, 0, len(counters)*2)
		for i, m := range counters {
			base := i * 2
			params := fmt.Sprintf("($%d, $%d)", base+1, base+2)
			values = append(values, params)
			args = append(args, m.ID, *m.Delta)
		}

		query := `INSERT INTO counters (id, delta) VALUES ` + strings.Join(values, ",") + ` ON CONFLICT (id) DO UPDATE SET delta = counters.delta + EXCLUDED.delta`

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return err
			}
			return fmt.Errorf("failed to insert counter batch: %w", err)
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

	_, err := db.pool.Exec(ctx, query, metric.ID, *metric.Value)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return err
		}
		return fmt.Errorf("database error: failed to insert gauge metric: %w", err)
	}

	return nil
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

	_, err := db.pool.Exec(ctx, query, metric.ID, *metric.Delta)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return err
		}
		return fmt.Errorf("database error: failed to insert counter metric: %w", err)
	}

	return nil
}

func (db *DB) GetGauge(ctx context.Context, id string) (float64, error) {
	query := `
		SELECT value FROM gauges WHERE id = $1
	`

	row := db.pool.QueryRow(ctx, query, id)
	var value float64
	err := row.Scan(&value)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return 0, err
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrMetricNotFound
		}
		return 0, fmt.Errorf("database error: failed to get gauge metric: %w", err)
	}

	return value, nil
}

func (db *DB) GetCounter(ctx context.Context, id string) (int64, error) {
	query := `
		SELECT delta FROM counters WHERE id = $1
	`

	row := db.pool.QueryRow(ctx, query, id)
	var delta int64
	err := row.Scan(&delta)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return 0, err
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrMetricNotFound
		}
		return 0, fmt.Errorf("database error: failed to get counter metric: %w", err)
	}

	return delta, nil
}

func (db *DB) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	query := `
		SELECT id, value FROM gauges
	`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("database error: failed to execute query to get all gauge metrics: %w", err)
	}
	defer rows.Close()

	m := make(map[string]float64)
	for rows.Next() {
		var id string
		var value float64

		if err = rows.Scan(&id, &value); err != nil {
			return nil, fmt.Errorf("database error: failed to scan gauge metrics values from database: %w", err)
		}
		m[id] = value
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("database error: error occurred while iterating over gauge metrics: %w", err)
	}

	if len(m) == 0 {
		return nil, models.ErrMetricNotFound
	}

	return m, nil
}

func (db *DB) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT id, delta FROM counters
	`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("database error: failed to execute query to get all counter metrics: %w", err)
	}
	defer rows.Close()

	m := make(map[string]int64)
	for rows.Next() {
		var id string
		var delta int64

		if err = rows.Scan(&id, &delta); err != nil {
			return nil, fmt.Errorf("database error: failed to scan counter metrics values from database: %w", err)
		}
		m[id] = delta
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("database error: error occurred while iterating over counter metrics: %w", err)
	}

	if len(m) == 0 {
		return nil, models.ErrMetricNotFound
	}

	return m, nil
}
