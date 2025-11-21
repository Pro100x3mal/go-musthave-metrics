package configs

import (
	"encoding/json"
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

type JSONAgentConfig struct {
	PollInterval   string `json:"poll_interval"`
	ReportInterval string `json:"report_interval"`
	ServerAddr     string `json:"address"`
	LogLevel       string `json:"log_level"`
	Key            string `json:"signing_key"`
	RateLimit      *int   `json:"rate_limit"`
	PublicKeyPath  string `json:"crypto_key"`
}

const (
	defaultServerAddr = "localhost:8080"
	defaultLogLevel   = "info"
	defaultRateLimit  = 5
	defaultPollSec    = 2
	defaultReportSec  = 10
)

func GetConfig() (*AgentConfig, error) {
	var (
		pollSec, reportSec int
		cfg                AgentConfig
		configFilePath     string
	)

	cfg.ServerAddr = defaultServerAddr
	cfg.LogLevel = defaultLogLevel
	cfg.RateLimit = defaultRateLimit
	pollSec = defaultPollSec
	reportSec = defaultReportSec

	var (
		flagPollInterval   int
		flagReportInterval int
		flagServerAddr     string
		flagLogLevel       string
		flagKey            string
		flagRateLimit      int
		flagPublicKeyPath  string
	)

	flag.StringVar(&flagServerAddr, "a", "", "address of HTTP server")
	flag.IntVar(&flagPollInterval, "p", -1, "polling interval in seconds")
	flag.IntVar(&flagReportInterval, "r", -1, "reporting interval in seconds")
	flag.StringVar(&flagLogLevel, "log-level", "", "log level")
	flag.StringVar(&flagKey, "k", "", "signing key")
	flag.IntVar(&flagRateLimit, "l", -1, "report rate limit")
	flag.StringVar(&flagPublicKeyPath, "crypto-key", "", "path to public key file")
	flag.StringVar(&configFilePath, "config", "", "path to JSON config file")
	flag.StringVar(&configFilePath, "c", "", "path to JSON config file")
	flag.Parse()

	if configFilePath == "" {
		if envConfigFilePath, ok := os.LookupEnv("CONFIG"); ok && envConfigFilePath != "" {
			configFilePath = envConfigFilePath
		}
	}

	if configFilePath != "" {
		if err := loadJSONConfig(configFilePath, &cfg, &pollSec, &reportSec); err != nil {
			return nil, fmt.Errorf("failed to load JSON config: %w", err)
		}
	}

	if flagServerAddr != "" {
		cfg.ServerAddr = flagServerAddr
	}
	if flagLogLevel != "" {
		cfg.LogLevel = flagLogLevel
	}
	if flagPollInterval >= 0 {
		pollSec = flagPollInterval
	}
	if flagReportInterval >= 0 {
		reportSec = flagReportInterval
	}
	if flagKey != "" {
		cfg.Key = flagKey
	}
	if flagRateLimit >= 0 {
		cfg.RateLimit = flagRateLimit
	}
	if flagPublicKeyPath != "" {
		cfg.PublicKeyPath = flagPublicKeyPath
	}

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

	cfg.PollInterval = time.Duration(pollSec) * time.Second
	cfg.ReportInterval = time.Duration(reportSec) * time.Second

	return &cfg, nil
}

func loadJSONConfig(path string, cfg *AgentConfig, pollSec *int, reportSec *int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var jsonCfg JSONAgentConfig
	if err = json.Unmarshal(data, &jsonCfg); err != nil {
		return fmt.Errorf("failed to parse JSON config: %w", err)
	}

	if jsonCfg.ServerAddr != "" {
		cfg.ServerAddr = jsonCfg.ServerAddr
	}
	if jsonCfg.LogLevel != "" {
		cfg.LogLevel = jsonCfg.LogLevel
	}
	if jsonCfg.PollInterval != "" {
		duration, err := time.ParseDuration(jsonCfg.PollInterval)
		if err != nil {
			return fmt.Errorf("failed to parse poll_interval: %w", err)
		}
		cfg.PollInterval = duration
		*pollSec = int(duration.Seconds())
	}
	if jsonCfg.ReportInterval != "" {
		duration, err := time.ParseDuration(jsonCfg.ReportInterval)
		if err != nil {
			return fmt.Errorf("failed to parse report_interval: %w", err)
		}
		cfg.ReportInterval = duration
		*reportSec = int(duration.Seconds())
	}
	if jsonCfg.PublicKeyPath != "" {
		cfg.PublicKeyPath = jsonCfg.PublicKeyPath
	}
	if jsonCfg.Key != "" {
		cfg.Key = jsonCfg.Key
	}
	if jsonCfg.RateLimit != nil {
		cfg.RateLimit = *jsonCfg.RateLimit
	}

	return nil
}
