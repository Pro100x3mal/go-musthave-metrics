package config

import (
	"flag"
	"time"
)

type AgentConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func GetConfig() AgentConfig {
	var pollSec, reportSec int

	flag.IntVar(&pollSec, "p", 2, "polling interval in seconds")
	flag.IntVar(&reportSec, "r", 10, "reporting interval in seconds")
	flag.Parse()

	return AgentConfig{
		PollInterval:   time.Duration(pollSec) * time.Second,
		ReportInterval: time.Duration(reportSec) * time.Second,
	}
}
