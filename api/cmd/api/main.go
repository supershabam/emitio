package main

import (
	"context"
	"net/http"

	"github.com/supershabam/emitio/api/v1"
	"go.uber.org/zap"
)

func main() {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(l)
	ctx := context.Background()
	s, err := v1.NewServer(ctx, v1.WithHistogrammer(&v1.Filesystem{Root: "./"}))
	if err != nil {
		panic(err)
	}
	zap.L().Info("listening", zap.String("addr", ":8080"))
	err = http.ListenAndServe(":8080", s)
	if err != nil {
		zap.L().Fatal("exiting from listen and serve", zap.Error(err))
	}
}
