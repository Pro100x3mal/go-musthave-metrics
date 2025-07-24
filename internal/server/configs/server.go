package configs

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	ServerAddr    string
	LogLevel      string
	StoreInterval time.Duration
	FileStorePath string
	IsRestore     bool
}

func GetConfig() *ServerConfig {
	var (
		storeInterval int
		cfg           ServerConfig
	)

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.IntVar(&storeInterval, "i", 300, "store interval in seconds")
	flag.StringVar(&cfg.FileStorePath, "f", "file_storage.json", "path to metrics storage file")
	flag.BoolVar(&cfg.IsRestore, "r", false, "load metrics from file on startup")

	flag.Parse()

	if envServerAddr := os.Getenv("ADDRESS"); envServerAddr != "" {
		cfg.ServerAddr = envServerAddr
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		var err error
		storeInterval, err = strconv.Atoi(envStoreInterval)
		if err != nil {
			log.Fatalf("invalid STORE_INTERVAL flag: %v", err)
		}
	}
	cfg.StoreInterval = time.Duration(storeInterval) * time.Second

	if envFileStorePath := os.Getenv("FILE_STORE_PATH"); envFileStorePath != "" {
		cfg.FileStorePath = envFileStorePath
	}

	if envIsRestore := os.Getenv("RESTORE"); envIsRestore != "" {
		var err error
		cfg.IsRestore, err = strconv.ParseBool(envIsRestore)
		if err != nil {
			log.Fatalf("invalid RESTORE flag: %v", err)
		}
	}

	return &cfg
}
