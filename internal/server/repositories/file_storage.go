package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
	"go.uber.org/zap"
)

type FileStorage struct {
	*MemStorage
	logger    *zap.Logger
	file      *os.File
	cfg       *configs.ServerConfig
	fileMutex *sync.Mutex
	isSync    bool
}

func NewFileStorage(ctx context.Context, cfg *configs.ServerConfig, ms *MemStorage, wg *sync.WaitGroup, logger *zap.Logger) (*FileStorage, error) {
	fsLogger := logger.With(zap.String("component", "file_storage"))
	fsLogger.Info("initializing file storage", zap.String("path", cfg.FileStoragePath))
	file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		fsLogger.Error("error opening storage file", zap.String("path", cfg.FileStoragePath), zap.Error(err))
		return nil, err
	}

	fs := &FileStorage{
		MemStorage: ms,
		logger:     fsLogger,
		file:       file,
		cfg:        cfg,
		fileMutex:  &sync.Mutex{},
	}

	if cfg.IsRestore {
		fs.logger.Info("restoring metrics from file", zap.String("path", cfg.FileStoragePath))
		if err = fs.restore(); err != nil {
			fs.logger.Error("failed to restore metrics from file", zap.Error(err))
			return nil, err
		}
		fs.logger.Info("metrics restored successfully")
	}

	if cfg.StoreInterval == 0 {
		fs.isSync = true
	}

	if cfg.StoreInterval > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fs.runSaveByInterval(ctx, cfg.StoreInterval)
		}()
	}

	return fs, nil
}

func (fs *FileStorage) runSaveByInterval(ctx context.Context, interval time.Duration) {
	fs.logger.Info("starting auto save loop")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fs.logger.Info("running auto save loop")

			fs.MemStorage.mu.RLock()
			if err := fs.save(); err != nil {
				fs.logger.Error("failed to save metrics to file", zap.Error(err))
			}
			fs.MemStorage.mu.RUnlock()
		case <-ctx.Done():
			fs.logger.Info("stopping auto save loop")

			fs.MemStorage.mu.RLock()
			if err := fs.save(); err != nil {
				fs.logger.Error("failed to save metrics to file on shutdown", zap.Error(err))
			}
			fs.MemStorage.mu.RUnlock()

			if err := fs.close(); err != nil {
				fs.logger.Error("failed to close file storage", zap.Error(err))
			}
			return
		}
	}
}

func (fs *FileStorage) restore() error {
	if _, err := fs.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to beginning of file %q: %w", fs.file.Name(), err)
	}

	data, err := io.ReadAll(fs.file)
	if err != nil {
		return fmt.Errorf("failed to read data from file %q: %w", fs.file.Name(), err)
	}

	if len(data) == 0 {
		return nil
	}

	var list []*models.Metrics
	if err = json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("failed to decode metrics JSON from file %q: %w", fs.file.Name(), err)
	}

	for _, metric := range list {
		switch metric.MType {
		case models.Gauge:
			err = fs.MemStorage.UpdateGauge(metric)
			if err != nil {
				return fmt.Errorf("failed to update %s metric %q: %w", metric.MType, metric.ID, err)
			}
		case models.Counter:
			err = fs.MemStorage.UpdateCounter(metric)
			if err != nil {
				return fmt.Errorf("failed to update %s metric %q: %w", metric.MType, metric.ID, err)
			}
		}
	}
	return nil
}

func (fs *FileStorage) save() error {
	fs.fileMutex.Lock()
	defer fs.fileMutex.Unlock()

	var list []*models.Metrics

	for name, value := range fs.MemStorage.gauges {
		v := value
		list = append(list, &models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		})
	}
	for name, delta := range fs.MemStorage.counters {
		d := delta
		list = append(list, &models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &d,
		})
	}

	data, err := json.MarshalIndent(list, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to JSON: %w", err)
	}

	err = fs.file.Truncate(0)
	if err != nil {
		return fmt.Errorf("failed to truncate file %q: %w", fs.file.Name(), err)
	}

	_, err = fs.file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to beginning of file %q: %w", fs.file.Name(), err)
	}

	_, err = fs.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write metrics to file %q: %w", fs.file.Name(), err)
	}

	return nil
}

func (fs *FileStorage) close() error {
	return fs.file.Close()
}

func (fs *FileStorage) UpdateGauge(metric *models.Metrics) error {
	if err := fs.MemStorage.UpdateGauge(metric); err != nil {
		return err
	}

	if fs.isSync {
		return fs.save()
	}
	return nil
}

func (fs *FileStorage) UpdateCounter(metric *models.Metrics) error {
	if err := fs.MemStorage.UpdateCounter(metric); err != nil {
		return err
	}

	if fs.isSync {
		return fs.save()
	}
	return nil
}
