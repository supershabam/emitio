package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sort"
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
	a.buckets["le+inf"] = (a.buckets["le+inf"] || 0) + 1	
    if (line.length <= 80) {
		a.buckets["le80"] = (a.buckets["le80"] || 0) + 1
	} 
	if (line.length <= 160) {
		a.buckets["le160"] = (a.buckets["le160"] || 0) + 1
	}	
	if (line.length <= 320) {
		a.buckets["le320"] = (a.buckets["le320"] || 0) + 1
	} 
	if (line.length <= 640) {
		a.buckets["le640"] = (a.buckets["le640"] || 0) + 1
	}
	if (line.length > 640) {
		a["outliers"] = a["outliers"] || []
		var l = JSON.parse(line)
		// capture line's sequence id
		a["outliers"].push(l.s)
	}
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
				keys := []string{}
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				fmt.Println("buckets")
				for _, k := range keys {
					v := m[k]
					fmt.Printf("%s: \t%d\n", k, v)
				}
				buckets = m
			}
		}
	})
	err = eg.Wait()
	if err != nil {
		logger.Fatal("error executing", zap.Error(err))
	}
}
