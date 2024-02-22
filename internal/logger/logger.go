package logger

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func New(loggingLevel string) (*zap.Logger, error) {
	level, err := zap.ParseAtomicLevel(loggingLevel)
	if err != nil {
		return nil, errors.Wrap(err, "parse atomic level")
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.Level = level

	return cfg.Build()
}
