package configs

import (
	"flag"
	"os"
)

type ServerConfig struct {
	ServerAddr string
	LogLevel   string
}

func GetConfig() *ServerConfig {
	cfg := ServerConfig{}
	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of HTTP server")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")

	flag.Parse()

	if envServerAddr := os.Getenv("ADDRESS"); envServerAddr != "" {
		cfg.ServerAddr = envServerAddr
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	return &cfg
}
