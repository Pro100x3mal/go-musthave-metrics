package config

import (
	"flag"
)

type ServerConfig struct {
	ServerAddr string
}

func GetConfig() ServerConfig {
	cfg := ServerConfig{}
	flag.StringVar(&cfg.ServerAddr, "addr", "localhost:8080", "address of HTTP server")

	flag.Parse()
	return cfg
}
