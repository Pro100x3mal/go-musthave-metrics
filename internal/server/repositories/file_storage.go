package repositories

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/models"
)

type FileStorage struct {
	*MemStorage
	file      *os.File
	cfg       *configs.ServerConfig
	fileMutex *sync.Mutex
}

func NewFileStorage(cfg *configs.ServerConfig, ms *MemStorage) (*FileStorage, error) {
	file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, fmt.Errorf("error opening storage file %s: %w", cfg.FileStoragePath, err)
	}

	return &FileStorage{
		MemStorage: ms,
		file:       file,
		cfg:        cfg,
		fileMutex:  &sync.Mutex{},
	}, nil
}

func (fs *FileStorage) Close() error {
	return fs.file.Close()
}

func (fs *FileStorage) SaveToFile() error {
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

func (fs *FileStorage) Restore() error {
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
			err = fs.UpdateGauge(metric)
			if err != nil {
				return fmt.Errorf("failed to update %s metric %q: %w", metric.MType, metric.ID, err)
			}
		case models.Counter:
			err = fs.UpdateCounter(metric)
			if err != nil {
				return fmt.Errorf("failed to update %s metric %q: %w", metric.MType, metric.ID, err)
			}
		}
	}
	return nil
}

func (fs *FileStorage) UpdateGauge(metric *models.Metrics) error {
	if err := fs.MemStorage.UpdateGauge(metric); err != nil {
		return err
	}

	if fs.cfg.StoreInterval == 0 {
		return fs.SaveToFile()
	}
	return nil
}

func (fs *FileStorage) UpdateCounter(metric *models.Metrics) error {
	if err := fs.MemStorage.UpdateCounter(metric); err != nil {
		return err
	}

	if fs.cfg.StoreInterval == 0 {
		return fs.SaveToFile()
	}
	return nil
}
