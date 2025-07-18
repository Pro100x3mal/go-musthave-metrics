package configs

import (
	"flag"
	"os"
)

type ServerConfig struct {
	ServerAddr string
}

func GetConfig() ServerConfig {
	cfg := ServerConfig{}
	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of HTTP server")

	flag.Parse()

	if envServerAddr := os.Getenv("ADDRESS"); envServerAddr != "" {
		cfg.ServerAddr = envServerAddr
	}
	return cfg
}
