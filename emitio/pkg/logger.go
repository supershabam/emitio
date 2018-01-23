package pkg

import (
	"go.uber.org/zap"
)

func ParseLogger(rawuri string) (*zap.Logger, error) {
	return zap.NewProduction()
}
