package pkg

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

func ParseLogger(rawuri string) (*zap.Logger, error) {
	return zap.NewProduction()
}

type loggerKey struct{}

var _loggerKey loggerKey

func SetLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, _loggerKey, logger)
}

func Logger(ctx context.Context) (*zap.Logger, error) {
	loggeri := ctx.Value(_loggerKey)
	logger, ok := loggeri.(*zap.Logger)
	if !ok {
		return nil, errors.New("logger not set on context")
	}
	return logger, nil
}

func MustLogger(ctx context.Context) *zap.Logger {
	logger, err := Logger(ctx)
	if err != nil {
		panic(err)
	}
	return logger
}
