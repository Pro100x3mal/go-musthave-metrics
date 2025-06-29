package main

import (
	"log"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/config"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/handler"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/repository"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetConfig()
	repo := repository.NewMemStorage()
	metricsService := service.NewMetricsService(repo)

	return handler.Serve(cfg, metricsService)
}
