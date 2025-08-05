package configs

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	ServerAddr      string
	LogLevel        string
	StoreInterval   time.Duration
	FileStoragePath string
	IsRestore       bool
	DatabaseDSN     string
}

func GetConfig() (*ServerConfig, error) {
	var (
		storeInterval int
		cfg           ServerConfig
	)

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.IntVar(&storeInterval, "i", 300, "store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "path to metrics storage file")
	flag.BoolVar(&cfg.IsRestore, "r", false, "load metrics from file on startup")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database PostgreSQL DSN")

	flag.Parse()

	if envServerAddr, ok := os.LookupEnv("ADDRESS"); ok && envServerAddr != "" {
		cfg.ServerAddr = envServerAddr
	}

	if envLogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok && envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok && envStoreInterval != "" {
		var err error
		storeInterval, err = strconv.Atoi(envStoreInterval)
		if err != nil {
			return nil, fmt.Errorf("failed to parse STORE_INTERVAL value '%s' to integer: %w", envStoreInterval, err)
		}
		if storeInterval < 0 {
			return nil, fmt.Errorf("STORE_INTERVAL value '%s' must be greater than 0", envStoreInterval)
		}
	}
	cfg.StoreInterval = time.Duration(storeInterval) * time.Second

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok && envFileStoragePath != "" {
		cfg.FileStoragePath = envFileStoragePath
	}

	if envIsRestore, ok := os.LookupEnv("RESTORE"); ok && envIsRestore != "" {
		var err error
		cfg.IsRestore, err = strconv.ParseBool(envIsRestore)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RESTORE value '%s' to boolean: %w", envIsRestore, err)
		}
	}

	if envDatabaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok && envDatabaseDSN != "" {
		cfg.DatabaseDSN = envDatabaseDSN
	}

	return &cfg, nil
}
