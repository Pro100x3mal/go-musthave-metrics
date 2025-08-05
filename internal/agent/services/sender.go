package services

import (
	"bytes"
	"compress/gzip"
	"encoding/json"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type RepositoryReader interface {
	GetAllMetrics() []*models.Metrics
}

type MetricsQueryService struct {
	reader RepositoryReader
	logger *zap.Logger
}

func NewMetricsQueryService(reader RepositoryReader, logger *zap.Logger) *MetricsQueryService {
	return &MetricsQueryService{
		reader: reader,
		logger: logger,
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
	//metrics := qs.reader.GetAllMetrics()
	//
	//for _, m := range metrics {
	//	var valueStr string
	//
	//	switch m.MType {
	//	case models.Gauge:
	//		if m.Value == nil {
	//			continue
	//		}
	//		valueStr = strconv.FormatFloat(*m.Value, 'f', -1, 64)
	//	case models.Counter:
	//		if m.Delta == nil {
	//			continue
	//		}
	//		valueStr = strconv.FormatInt(*m.Delta, 10)
	//	default:
	//		continue
	//	}
	//
	//	qs.logger.Info("sending metric",
	//		zap.String("type", m.MType),
	//		zap.String("id", m.ID),
	//		zap.String("value", valueStr),
	//	)
	//
	//	buf := &bytes.Buffer{}
	//	gz := gzip.NewWriter(buf)
	//	err := json.NewEncoder(gz).Encode(m)
	//	if err != nil {
	//		qs.logger.Error("gzip encoding failed", zap.Error(err))
	//		continue
	//	}
	//	if err = gz.Close(); err != nil {
	//		qs.logger.Error("failed to close gzip writer", zap.Error(err))
	//		continue
	//	}
	//
	//	_, err = c.client.R().
	//		SetHeader("Content-Encoding", "gzip").
	//		SetHeader("Content-Type", "application/json").
	//		SetBody(buf.Bytes()).
	//		Post("/update")
	//
	//	if err != nil {
	//		qs.logger.Info("could not post metric to server",
	//			zap.String("type", m.MType),
	//			zap.String("id", m.ID),
	//			zap.String("value", valueStr),
	//			zap.String("url", "/update"),
	//			zap.Error(err),
	//		)
	//		continue
	//	}
	//
	//}
	//
	//qs.logger.Info("all metrics was sent succesfully")

	metrics := qs.reader.GetAllMetrics()
	if len(metrics) == 0 {
		qs.logger.Info("no metrics to send")
		return
	}

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	err := json.NewEncoder(gz).Encode(metrics)
	if err != nil {
		qs.logger.Error("gzip encoding failed", zap.Error(err))
	}
	if err = gz.Close(); err != nil {
		qs.logger.Error("failed to close gzip writer", zap.Error(err))
	}

	_, err = c.client.R().
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Content-Type", "application/json").
		SetBody(buf.Bytes()).
		Post("/updates/")

	if err != nil {
		qs.logger.Info("could not post metric to server",
			zap.String("url", "/updates"),
			zap.Error(err),
		)
	}

	qs.logger.Info("all metrics was sent succesfully")
}
