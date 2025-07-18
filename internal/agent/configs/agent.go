package configs

import (
	"flag"
	"log"
	"os"
	"strconv"
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

	if envServerAddr := os.Getenv("ADDRESS"); envServerAddr != "" {
		cfg.ServerAddr = envServerAddr
	}

	if envPollSecStr := os.Getenv("POLL_INTERVAL"); envPollSecStr != "" {
		envPollSecInt, err := strconv.Atoi(envPollSecStr)
		if err != nil {
			log.Fatalf("invalid POLL_INTERVAL: %v", err)
		}
		pollSec = envPollSecInt
	}

	if envReportSecStr := os.Getenv("REPORT_INTERVAL"); envReportSecStr != "" {
		envReportSecInt, err := strconv.Atoi(envReportSecStr)
		if err != nil {
			log.Fatalf("invalid REPORT_INTERVAL: %v", err)
		}
		reportSec = envReportSecInt
	}

	cfg.PollInterval = time.Duration(pollSec) * time.Second
	cfg.ReportInterval = time.Duration(reportSec) * time.Second

	return cfg
}
