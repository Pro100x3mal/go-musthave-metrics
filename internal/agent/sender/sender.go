package sender

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/model"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/service"
)

func SendMetrics(repo service.MetricsRepository) {
	client := &http.Client{}
	metrics := repo.GetAllMetrics()

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

		url := fmt.Sprintf("http://localhost:8080/update/%s/%s/%s", m.MType, m.ID, valueStr)
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			log.Println("request error:", err)
			continue
		}
		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			log.Println("post error:", err)
			continue
		}
		resp.Body.Close()
	}
}
