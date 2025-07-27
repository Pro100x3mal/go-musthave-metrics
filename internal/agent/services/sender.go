package services

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/logger"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type RepositoryReader interface {
	GetAllMetrics() []*models.Metrics
}

type MetricsQueryService struct {
	reader RepositoryReader
}

func NewMetricsQueryService(reader RepositoryReader) *MetricsQueryService {
	return &MetricsQueryService{
		reader: reader,
	}
}

type Client struct {
	client *resty.Client
}

func NewClient(cfg *configs.AgentConfig) *Client {
	return &Client{
		client: resty.New().
			SetBaseURL("http://" + cfg.ServerAddr),
	}
}

func (qs *MetricsQueryService) SendMetrics(c *Client) {
	metrics := qs.reader.GetAllMetrics()

	for _, m := range metrics {
		var valueStr string

		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				continue
			}
			valueStr = strconv.FormatFloat(*m.Value, 'f', -1, 64)
		case models.Counter:
			if m.Delta == nil {
				continue
			}
			valueStr = strconv.FormatInt(*m.Delta, 10)
		default:
			continue
		}

		logger.Log.Info("sending metric",
			zap.String("type", m.MType),
			zap.String("id", m.ID),
			zap.String("value", valueStr),
		)

		buf := &bytes.Buffer{}
		gz := gzip.NewWriter(buf)
		err := json.NewEncoder(gz).Encode(m)
		if err != nil {
			logger.Log.Error("gzip encoding failed", zap.Error(err))
			continue
		}
		if err = gz.Close(); err != nil {
			logger.Log.Error("failed to close gzip writer", zap.Error(err))
			continue
		}

		_, err = c.client.R().
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Content-Type", "application/json").
			SetBody(buf.Bytes()).
			Post("/update")

		if err != nil {
			logger.Log.Info("could not post metric to server",
				zap.String("type", m.MType),
				zap.String("id", m.ID),
				zap.String("value", valueStr),
				zap.String("url", "/update"),
				zap.Error(err),
			)
			continue
		}

	}

	logger.Log.Info("all metrics was sent succesfully")
}
