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
}

func GetConfig() (*ServerConfig, error) {
	var (
		storeInterval int
		cfg           ServerConfig
	)

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.IntVar(&storeInterval, "i", 300, "store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", "file_storage.json", "path to metrics storage file")
	flag.BoolVar(&cfg.IsRestore, "r", false, "load metrics from file on startup")

	flag.Parse()

	if envServerAddr, exist := os.LookupEnv("ADDRESS"); exist {
		if envServerAddr != "" {
			cfg.ServerAddr = envServerAddr
		}
	}

	if envLogLevel, exist := os.LookupEnv("LOG_LEVEL"); exist {
		if envLogLevel != "" {
			cfg.LogLevel = envLogLevel
		}
	}

	if envStoreInterval, exist := os.LookupEnv("STORE_INTERVAL"); exist {
		if envStoreInterval != "" {
			var err error
			storeInterval, err = strconv.Atoi(envStoreInterval)
			if err != nil {
				return nil, fmt.Errorf("failed to parse STORE_INTERVAL value '%s' to integer: %w", envStoreInterval, err)
			}
			if storeInterval < 0 {
				return nil, fmt.Errorf("STORE_INTERVAL value '%s' must be greater than 0", envStoreInterval)
			}
		}
	}
	cfg.StoreInterval = time.Duration(storeInterval) * time.Second

	if envFileStoragePath, exist := os.LookupEnv("FILE_STORAGE_PATH"); exist {
		if envFileStoragePath != "" {
			cfg.FileStoragePath = envFileStoragePath
		}
	}

	if envIsRestore, exist := os.LookupEnv("RESTORE"); exist {
		if envIsRestore != "" {
			var err error
			cfg.IsRestore, err = strconv.ParseBool(envIsRestore)
			if err != nil {
				return nil, fmt.Errorf("failed to parse RESTORE value '%s' to boolean: %w", envIsRestore, err)
			}
		}
	}

	return &cfg, nil
}
