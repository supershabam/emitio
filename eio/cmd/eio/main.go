package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/supershabam/emitio/eio/pb/edge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const js = `
function transform(acc, line) {
	var a = JSON.parse(acc)
	a["buckets"] = (a["buckets"] || {})
	var key = ""
	if (line.length <= 10) {
		key = "lte10"
	} else if (line.length <= 20) {
		key = "lte20"
	} else if (line.length <= 40) {
		key = "lte40"
	} else if (line.length <= 80) {
		key = "lte80"
	} else if (line.length <= 160) {
		key = "lte160"
	} else if (line.length <= 320) {
		key = "lte320"
	} else if (line.length <= 640) {
		key = "lte640"
	} else {
		key = "lte+inf"
		a["outliers"] = a["outliers"] || []
		var l = JSON.parse(line)
		// capture line's sequence id
		a["outliers"].push(l.s)
	}
	a.buckets[key] = (a.buckets[key] || 0) + 1
	return [JSON.stringify(a), []]
}
`

func main() {
	ctx, cancel := context.WithCancel(context.Background())
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
	cc, err := grpc.DialContext(ctx, "138.197.67.130:9090", grpc.WithInsecure())
	if err != nil {
		logger.Fatal("dialing grpc", zap.Error(err))
	}
	ec := edge.NewEdgeClient(cc)
	reply, err := ec.Nodes(ctx, &edge.NodesRequest{})
	if err != nil {
		logger.Fatal("getting nodes", zap.Error(err))
	}
	logger.Info("got nodes", zap.Int("node_count", len(reply.Nodes)))
	clients := []*client{}
	eg, ctx := errgroup.WithContext(ctx)
	for _, node := range reply.Nodes {
		c := &client{
			buckets: map[string]int64{},
			c:       ec,
			node:    node,
			js:      js,
		}
		clients = append(clients, c)
		eg.Go(func() error {
			return c.run(ctx)
		})
	}
	eg.Go(func() error {
		buckets := map[string]int64{}
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				m := map[string]int64{}
				for _, c := range clients {
					c.sumTo(m)
				}
				if reflect.DeepEqual(m, buckets) {
					continue
				}
				fmt.Printf("buckets: %+v\n", m)
				buckets = m
			}
		}
	})
	err = eg.Wait()
	if err != nil {
		logger.Fatal("error executing", zap.Error(err))
	}
}
