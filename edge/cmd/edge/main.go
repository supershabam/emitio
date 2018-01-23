package main

import (
	"context"
	"crypto/tls"
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
		apiAddr = ":9090"
	)
	cer, err := tls.LoadX509KeyPair(
		"/etc/letsencrypt/live/edge.emit.io/fullchain.pem",
		"/etc/letsencrypt/live/edge.emit.io/privkey.pem",
	)
	if err != nil {
		zap.L().Fatal("while loading x509 key pair", zap.Error(err))
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	rgrpcL, err := tls.Listen("tcp", ":443", config)
	if err != nil {
		zap.L().Fatal("tls listen", zap.Error(err))
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
