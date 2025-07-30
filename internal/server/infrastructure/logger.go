package infrastructure

import (
	"fmt"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"go.uber.org/zap"
)

func NewLogger(cfg *configs.ServerConfig) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	lConf := zap.NewDevelopmentConfig()
	lConf.Level = lvl

	zl, err := lConf.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger config: %w", err)
	}

	return zl, nil
}
