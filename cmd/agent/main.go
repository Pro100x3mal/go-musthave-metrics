package main

import (
	"log"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/config"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/repository"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/sender"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetConfig()
	repo := repository.NewMemStorage()
	collectService := service.NewMetricsCollectService(repo)
	queryService := service.NewMetricsQueryService(repo)

	tickerPoll := time.NewTicker(cfg.PollInterval)
	tickerReport := time.NewTicker(cfg.ReportInterval)
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	go func() {
		for range tickerPoll.C {
			err := collectService.CollectMetrics()
			if err != nil {
				log.Println("collect error:", err)
			}
		}
	}()

	go func() {
		for range tickerReport.C {
			sender.SendMetrics(queryService, cfg)
		}
	}()

	select {}
}
