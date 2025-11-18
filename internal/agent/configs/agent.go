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
	Key            string
	RateLimit      int
	PublicKeyPath  string
}

func GetConfig() (*AgentConfig, error) {
	var (
		pollSec, reportSec int
		cfg                AgentConfig
	)

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.IntVar(&pollSec, "p", 2, "polling interval in seconds")
	flag.IntVar(&reportSec, "r", 10, "reporting interval in seconds")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "log level")
	flag.StringVar(&cfg.Key, "k", "", "signing key")
	flag.IntVar(&cfg.RateLimit, "l", 5, "report rate limit")
	flag.StringVar(&cfg.PublicKeyPath, "crypto-key", "", "path to public key file")
	flag.Parse()

	if envServerAddr, ok := os.LookupEnv("ADDRESS"); ok && envServerAddr != "" {
		cfg.ServerAddr = envServerAddr
	}

	if envLogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok && envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if envPollSecStr, ok := os.LookupEnv("POLL_INTERVAL"); ok && envPollSecStr != "" {
		envPollSecInt, err := strconv.Atoi(envPollSecStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse POLL_INTERVAL value %q to integer: %w", envPollSecStr, err)
		}
		pollSec = envPollSecInt
	}

	if envReportSecStr, ok := os.LookupEnv("REPORT_INTERVAL"); ok && envReportSecStr != "" {
		envReportSecInt, err := strconv.Atoi(envReportSecStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse REPORT_INTERVAL value %q to integer: %w", envReportSecStr, err)
		}
		reportSec = envReportSecInt
	}

	cfg.PollInterval = time.Duration(pollSec) * time.Second
	cfg.ReportInterval = time.Duration(reportSec) * time.Second

	if envKey, ok := os.LookupEnv("KEY"); ok && envKey != "" {
		cfg.Key = envKey
	}

	if envRateLimitStr, ok := os.LookupEnv("RATE_LIMIT"); ok && envRateLimitStr != "" {
		envRateLimit, err := strconv.Atoi(envRateLimitStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RATE_LIMIT value %q to integer: %w", envRateLimit, err)
		}
		cfg.RateLimit = envRateLimit
	}

	if envPublicKeyPath, ok := os.LookupEnv("CRYPTO_KEY"); ok && envPublicKeyPath != "" {
		cfg.PublicKeyPath = envPublicKeyPath
	}

	return &cfg, nil
}
