package main

import (
	"os"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/services"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg := configs.GetConfig()

	log, err := infrastructure.NewLogger(cfg)
	if err != nil {
		return err
	}
	defer log.Sync()

	repo := repositories.NewMemStorage()
	metricsService := services.NewMetricsService(repo)
	metricsHandler := handlers.NewMetricsHandler(metricsService)

	log.Info("starting application")

	if err = handlers.StartServer(cfg, log, metricsHandler); err != nil {
		return err
	}

	return nil
}
