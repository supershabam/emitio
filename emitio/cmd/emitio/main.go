package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/supershabam/emitio/emitio/pb/emitio"
	"github.com/supershabam/emitio/emitio/pkg"
	"github.com/supershabam/emitio/emitio/pkg/ingresses"
	"github.com/supershabam/emitio/emitio/pkg/listeners"
)

func init() {
	runtime.LockOSThread()
}

// --origin pod=$(pod_name)
// --origin namespace=$(k8s_namespace)
// --origin datacenter=nyc2
// --ingress tail:///var/log/message
// --ingress syslog+udp://0.0.0.0:514/?my_tag=value
// --ingress ndjson+stdin:///
// --ingress statsd+udp://0.0.0.0:9001/?region=nyc2#application=something
// --ingress tail:///var/log/mongodb/mongodb.log#hint=mongodb-v3.18
// --ingress opentracing+udp://0.0.0.0:9002/
// --forward https://ingress.emit.io/
// --storage file:///tmp/emitio/
func main() {
	// flags
	pflag.StringArrayP("origin", "o", []string{}, "tags to include with ALL messages collected")
	ingressURIs := pflag.StringArrayP("ingress", "i", []string{}, "udp://127.0.0.1:9009")
	loggerURI := pflag.StringP("logger", "l", "stderr:///?level=info", "configure the logger")
	storageURI := pflag.StringP("storage", "s", "file:///tmp/emitio/", "configure storage destination")
	targetURI := pflag.StringP("target", "t", "https://edge.emit.io/", "where to connect to")
	pflag.Parse()
	// set up logging
	logger, err := pkg.ParseLogger(*loggerURI)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)
	defer logger.Sync()
	// set up context
	ctx, cancel := context.WithCancel(context.Background())
	sigch := make(chan os.Signal, 2)
	signal.Notify(sigch, os.Interrupt)
	go func() {
		<-sigch
		cancel()
		<-sigch
		os.Exit(1)
	}()
	// set up storage
	db, err := pkg.ParseStorage(*storageURI)
	if err != nil {
		zap.L().Fatal("parse storage", zap.Error(err))
	}
	defer db.Close()
	// set up ingresses
	il := []pkg.Ingresser{}
	if ingressURIs != nil {
		for _, uri := range *ingressURIs {
			i, err := ingresses.MakeIngress(uri)
			if err != nil {
				zap.L().Fatal("unable to make ingress", zap.String("ingress", uri))
			}
			il = append(il, i)
		}
	}
	// tags, err := pkg.ParseOriginTags(*originTags)
	// if err != nil {
	// 	logger.Fatal("parse origin tags", zap.Error(err))
	// }
	// create server
	s, err := pkg.NewServer(ctx,
		pkg.WithIngresses(il),
		pkg.WithDB(db),
		// pkg.withTags(tags),
	)
	if err != nil {
		zap.L().Fatal("new server", zap.Error(err))
	}
	lis, err := listeners.NewReverse(ctx, listeners.WithTarget(*targetURI))
	if err != nil {
		zap.L().Fatal("new reverse", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	emitio.RegisterEmitioServer(grpcServer, s)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.Run(ctx)
	})
	eg.Go(func() error {
		go func() {
			<-ctx.Done()
			timeout := 5 * time.Second
			zap.L().Info("shutting down gracefully with timeout", zap.Duration("timeout", timeout))
			tctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			go func() {
				<-tctx.Done()
				grpcServer.Stop()
			}()
			grpcServer.GracefulStop()
		}()
		return grpcServer.Serve(lis)
	})
	err = eg.Wait()
	if err != nil {
		zap.L().Fatal("unrecoverable error", zap.Error(err))
	}
}
