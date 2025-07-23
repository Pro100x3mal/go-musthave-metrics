package services

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/infrastructure"
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
		//SetHeader("Content-Type", "text/plain"),
	}
}

func (qs *MetricsQueryService) SendMetrics(c *Client, log *infrastructure.Logger) {
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

		log.Info("sending metric",
			zap.String("type", m.MType),
			zap.String("id", m.ID),
			zap.String("value", valueStr),
		)

		//_, err := c.client.R().
		//	SetPathParam("type", m.MType).
		//	SetPathParam("name", m.ID).
		//	SetPathParam("value", valueStr).
		//	Post("/update/{type}/{name}/{value}")

		buf := &bytes.Buffer{}
		gz := gzip.NewWriter(buf)
		err := json.NewEncoder(gz).Encode(m)
		if err != nil {
			log.Error("gzip encoding failed", zap.Error(err))
			continue
		}
		gz.Close()

		_, err = c.client.R().
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Content-Type", "application/json").
			//SetHeader("Accept-Encoding", "gzip").
			SetBody(buf.Bytes()).
			Post("/update")

		if err != nil {
			log.Error("could not post metric to server",
				zap.String("type", m.MType),
				zap.String("id", m.ID),
				zap.String("value", valueStr),
				//zap.String("url", "/update/"+m.MType+"/"+m.ID+"/"+valueStr),
				zap.String("url", "/update"),
				zap.Error(err),
			)
			continue
		}

	}

	log.Info("all metrics was sent succesfully")
}
