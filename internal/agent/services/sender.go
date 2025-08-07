package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"time"

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
			SetBaseURL("http://" + cfg.ServerAddr).
			SetTimeout(10 * time.Second).
			SetRetryCount(3).
			SetRetryWaitTime(1 * time.Second).
			SetRetryMaxWaitTime(5 * time.Second),
	}
}

func (qs *MetricsQueryService) SendMetrics(ctx context.Context, c *Client) {
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
		return
	}
	if err = gz.Close(); err != nil {
		qs.logger.Error("failed to close gzip writer", zap.Error(err))
		return
	}

	_, err = c.client.R().
		SetContext(ctx).
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Content-Type", "application/json").
		SetBody(buf.Bytes()).
		Post("/updates/")

	if err != nil {
		if ctx.Err() != nil {
			qs.logger.Info("request cancelled due to shutdown")
			return
		}

		qs.logger.Error("failed to send metrics", zap.Error(err))
	}

	qs.logger.Info("all metrics was sent succesfully")
}
