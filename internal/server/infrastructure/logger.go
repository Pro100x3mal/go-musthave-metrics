package infrastructure

import (
	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/configs"
	"go.uber.org/zap"
)

type Logger struct {
	*zap.Logger
}

func NewLogger(cfg *configs.ServerConfig) (*Logger, error) {
	lvl, err := zap.ParseAtomicLevel(cfg.LogLevel)
	if err != nil {
		fallback := zap.NewExample()
		fallback.Error("failed to initialize logger: invalid log level", zap.Error(err))
		return &Logger{fallback}, err
	}

	lConf := zap.NewDevelopmentConfig()
	lConf.Level = lvl

	zl, err := lConf.Build()
	if err != nil {
		fallback := zap.NewExample()
		fallback.Error("failed to initialize logger: failed to build logger config", zap.Error(err))
		return &Logger{fallback}, err
	}

	return &Logger{zl}, nil
}
