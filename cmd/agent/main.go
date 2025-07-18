package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/agent/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := configs.GetConfig()
	repo := repositories.NewMemStorage()
	collectService := services.NewMetricsCollectService(repo)
	queryService := services.NewMetricsQueryService(repo)

	newClient := services.NewClient(cfg)

	tickerPoll := time.NewTicker(cfg.PollInterval)
	tickerReport := time.NewTicker(cfg.ReportInterval)
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down gracefully...")
			return ctx.Err()
		case <-tickerPoll.C:
			if err := collectService.UpdateAllMetrics(); err != nil {
				log.Println("collect error:", err)
			}
		case <-tickerReport.C:
			queryService.SendMetrics(newClient)
			if err := collectService.ResetPollCount(); err != nil {
				log.Println("collect error:", err)
			}
		}
	}
}
