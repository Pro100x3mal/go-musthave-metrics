package sender

import (
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/config"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/model"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/service"
	"github.com/go-resty/resty/v2"
)

func SendMetrics(queryService *service.MetricsQueryService, cfg config.AgentConfig) {
	client := resty.New()
	metrics := queryService.GetAllMetrics()

	for _, m := range metrics {
		var valueStr string

		switch m.MType {
		case model.Gauge:
			if m.Value == nil {
				continue
			}
			valueStr = fmt.Sprintf("%f", *m.Value)
		case model.Counter:
			if m.Delta == nil {
				continue
			}
			valueStr = strconv.FormatInt(*m.Delta, 10)
		default:
			continue
		}

		u := url.URL{
			Scheme: "http",
			Host:   cfg.ServerAddr,
			Path:   fmt.Sprintf("/update/%s/%s/%s", m.MType, m.ID, valueStr),
		}

		resp, err := client.R().
			SetHeader("Content-Type", "text/plain").
			Post(u.String())

		if err != nil {
			log.Println("post error:", err)
			continue
		}
		resp.RawBody().Close()
	}
}
