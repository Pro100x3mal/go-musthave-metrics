package configs

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type AgentConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddr     string
	LogLevel       string
}

func GetConfig() (*AgentConfig, error) {
	var (
		pollSec, reportSec int
		cfg                AgentConfig
	)

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.IntVar(&pollSec, "p", 2, "polling interval in seconds")
	flag.IntVar(&reportSec, "r", 10, "reporting interval in seconds")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.Parse()

	if envServerAddr, exists := os.LookupEnv("ADDRESS"); exists {
		if envServerAddr != "" {
			cfg.ServerAddr = envServerAddr
		}
	}

	if envLogLevel, exist := os.LookupEnv("LOG_LEVEL"); exist {
		if envLogLevel != "" {
			cfg.LogLevel = envLogLevel
		}
	}

	if envPollSecStr, exist := os.LookupEnv("POLL_INTERVAL"); exist {
		if envPollSecStr != "" {
			envPollSecInt, err := strconv.Atoi(envPollSecStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse POLL_INTERVAL value '%s' to integer: %w", envPollSecStr, err)
			}
			pollSec = envPollSecInt
		}
	}

	if envReportSecStr, exist := os.LookupEnv("REPORT_INTERVAL"); exist {
		if envReportSecStr != "" {
			envReportSecInt, err := strconv.Atoi(envReportSecStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse REPORT_INTERVAL value '%s' to integer: %w", envReportSecStr, err)
			}
			reportSec = envReportSecInt
		}
	}

	cfg.PollInterval = time.Duration(pollSec) * time.Second
	cfg.ReportInterval = time.Duration(reportSec) * time.Second

	return &cfg, nil
}
