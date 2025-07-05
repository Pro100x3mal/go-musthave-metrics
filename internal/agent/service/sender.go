package service

import (
	"log"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/config"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/model"
	"github.com/go-resty/resty/v2"
)

type RepositoryReader interface {
	GetAllMetrics() []*model.Metrics
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

func NewClient(cfg config.AgentConfig) *Client {
	return &Client{
		client: resty.New().
			SetBaseURL("http://"+cfg.ServerAddr).
			SetHeader("Content-Type", "text/plain"),
	}
}

func (qs *MetricsQueryService) SendMetrics(c *Client) {
	metrics := qs.reader.GetAllMetrics()

	for _, m := range metrics {
		var valueStr string

		switch m.MType {
		case model.Gauge:
			if m.Value == nil {
				continue
			}
			valueStr = strconv.FormatFloat(*m.Value, 'f', -1, 64)
		case model.Counter:
			if m.Delta == nil {
				continue
			}
			valueStr = strconv.FormatInt(*m.Delta, 10)
		default:
			continue
		}

		log.Printf("sending metric: %-10s %-20s =%s", m.MType, m.ID, valueStr)

		_, err := c.client.R().
			SetPathParam("type", m.MType).
			SetPathParam("name", m.ID).
			SetPathParam("value", valueStr).
			Post("/update/{type}/{name}/{value}")

		if err != nil {
			log.Println("post error:", err)
			continue
		}

	}

	log.Printf("all metrics was sent succesfully")
}
