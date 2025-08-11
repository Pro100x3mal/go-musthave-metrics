package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/models"
	"github.com/go-resty/resty/v2"
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
			SetBaseURL("http://" + cfg.ServerAddr).
			SetTimeout(10 * time.Second).
			SetRetryCount(3).
			SetRetryWaitTime(1 * time.Second).
			SetRetryMaxWaitTime(5 * time.Second),
	}
}

func (qs *MetricsQueryService) SendMetrics(ctx context.Context, c *Client) error {
	metrics := qs.reader.GetAllMetrics()
	if len(metrics) == 0 {
		return errors.New("no metrics to send")
	}

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	err := json.NewEncoder(gz).Encode(metrics)
	if err != nil {
		return fmt.Errorf("gzip encoding failed: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	_, err = c.client.R().
		SetContext(ctx).
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Content-Type", "application/json").
		SetBody(buf.Bytes()).
		Post("/updates/")

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	return nil
}
