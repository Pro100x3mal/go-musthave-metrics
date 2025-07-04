package config

import (
	"flag"
	"time"
)

type AgentConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddr     string
}

func GetConfig() AgentConfig {
	var (
		pollSec, reportSec int
		cfg                AgentConfig
	)

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.IntVar(&pollSec, "p", 2, "polling interval in seconds")
	flag.IntVar(&reportSec, "r", 10, "reporting interval in seconds")
	flag.Parse()

	cfg.PollInterval = time.Duration(pollSec) * time.Second
	cfg.ReportInterval = time.Duration(reportSec) * time.Second

	return cfg
}
