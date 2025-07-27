package logger

import (
	"fmt"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"go.uber.org/zap"
)

var Log = zap.NewNop()

func Initialize(cfg *configs.ServerConfig) error {
	lvl, err := zap.ParseAtomicLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: invalid log level: %w", err)
	}

	lConf := zap.NewDevelopmentConfig()
	lConf.Level = lvl

	zl, err := lConf.Build()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: failed to build logger config: %w", err)
	}

	Log = zl
	return nil
}
