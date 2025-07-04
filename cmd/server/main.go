package main

import (
	"log"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/config"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handler"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repository"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetConfig()
	repo := repository.NewMemStorage()
	rReader, rWriter := repo, repo
	metricsService := service.NewMetricsService(rReader, rWriter)
	msReader, msWriter := metricsService, metricsService

	return handler.Serve(cfg, msReader, msWriter)
}
