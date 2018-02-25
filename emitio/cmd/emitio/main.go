package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/supershabam/emitio/emitio/pb/emitio"
	"github.com/supershabam/emitio/emitio/pkg"
	"github.com/supershabam/emitio/emitio/pkg/ingresses"
	"github.com/supershabam/emitio/emitio/pkg/listeners"
	"github.com/supershabam/emitio/emitio/pkg/storages"

	_ "github.com/mattn/go-sqlite3"
)

// --origin pod=$(pod_name)
// --origin namespace=$(k8s_namespace)
// --origin datacenter=nyc2
// --ingress follow:///var/log/message
// --ingress syslog+udp://0.0.0.0:514/?my_tag=value
// --ingress ndjson+stdin:///
// --ingress statsd+udp://0.0.0.0:9001/?region=nyc2#application=something
// --ingress follow:///var/log/mongodb/mongodb.log#hint=mongodb-v3.18
// --ingress opentracing+udp://0.0.0.0:9002/
// --forward https://ingress.emit.io/
// --storage file:///tmp/emitio/
// --web http://localhost:8080/
func main() {
	// flags
	pflag.StringArrayP("origin", "o", []string{}, "tags to include with ALL messages collected")
	ingressURIs := pflag.StringArrayP("ingress", "i", []string{}, "udp://127.0.0.1:9009")
	// loggerURI := pflag.StringP("logger", "l", "stderr:///?level=info", "configure the logger")
	storageURI := pflag.StringP("storage", "s", "memory://", "configure storage destination")
	targetURI := pflag.StringP("target", "t", "https://edge.emit.io/", "where to connect to")
	webURI := pflag.String("web", "http://localhost:9090/", "how to run a local webserver presenting a UI to diagnose a running emitio agent")
	pflag.Parse()
	// set up logging
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)
	defer logger.Sync()
	zap.L().Debug("greetings from debug")
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
	pr := prometheus.DefaultRegisterer
	ctx, err = pkg.InitContextMetrics(ctx, pr)
	if err != nil {
		zap.L().Fatal("initializing metrics", zap.Error(err))
	}
	// set up storage
	store, err := storages.NewSQLite(ctx,
		storages.WithResolverConfig(*storageURI),
	)
	if err != nil {
		zap.L().Fatal("parse storage", zap.Error(err))
	}
	// set up ingresses
	il := []pkg.Ingresser{}
	if ingressURIs != nil {
		for _, uri := range *ingressURIs {
			i, err := ingresses.MakeIngress(uri)
			if err != nil {
				zap.L().Fatal("unable to make ingress", zap.String("ingress", uri))
			}
			zap.L().Info("created ingress", zap.String("uri", uri))
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
		pkg.WithStorage(store),
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
	if len(*webURI) != 0 {
		u, err := url.Parse(*webURI)
		if err != nil {
			zap.L().Fatal("unable to parse web uri", zap.String("uri", *webURI), zap.Error(err))
		}
		httpsrv := &http.Server{
			Addr: u.Host,
		}
		http.Handle("/metrics", promhttp.Handler())
		eg.Go(func() error {
			zap.L().Info("serving telemetry", zap.String("addr", httpsrv.Addr))
			err := httpsrv.ListenAndServe()
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		})
		eg.Go(func() error {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			return httpsrv.Shutdown(shutdownCtx)
		})
	}
	err = eg.Wait()
	if err != nil {
		zap.L().Fatal("unrecoverable error", zap.Error(err))
	}
}
