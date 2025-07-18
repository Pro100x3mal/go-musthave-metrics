package main

import (
	"log"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/handlers"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/repositories"
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := configs.GetConfig()
	repo := repositories.NewMemStorage()

	receiverService := services.NewMetricsReceiverService(repo)
	queryService := services.NewMetricsQueryService(repo)

	return handlers.Serve(cfg, receiverService, queryService)
}
