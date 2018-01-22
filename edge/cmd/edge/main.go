package main

import (
	"context"
	"net"
	"os"
	"os/signal"

	"github.com/supershabam/emitio/edge/pkg"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigch := make(chan os.Signal, 2)
	signal.Notify(sigch, os.Interrupt)
	go func() {
		<-sigch
		cancel()
		<-sigch
		os.Exit(1)
	}()
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
	var (
		rgrpcAddr = ":8080"
		apiAddr   = ":9090"
	)
	zap.L().Info("reverse grpc listening", zap.String("addr", rgrpcAddr))
	rgrpcL, err := net.Listen("tcp", rgrpcAddr)
	if err != nil {
		zap.L().Fatal("rgrpc net listen", zap.Error(err))
	}
	zap.L().Info("api listening", zap.String("addr", apiAddr))
	apiL, err := net.Listen("tcp", apiAddr)
	if err != nil {
		zap.L().Fatal("api net listen", zap.Error(err))
	}
	s, err := pkg.NewServer(ctx, pkg.WithRGRPCListener(rgrpcL), pkg.WithAPIListener(apiL))
	if err != nil {
		zap.L().Fatal("creating server", zap.Error(err))
	}
	err = s.Run(ctx)
	if err != nil {
		zap.L().Fatal("running server", zap.Error(err))
	}
}
