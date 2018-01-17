package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"
	"github.com/supershabam/emitio/eio/pb/edge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const js = `
function transform(acc, lines) {
	return [acc, lines]
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
	client := edge.NewEdgeClient(cc)
	reply, err := client.GetNodes(ctx, &edge.GetNodesRequest{})
	if err != nil {
		logger.Fatal("getting nodes", zap.Error(err))
	}
	eg, ctx := errgroup.WithContext(ctx)
	for _, node := range reply.Nodes {
		node := node
		logger.Info("reading node", zap.String("node", node))
		eg.Go(func() error {
			t, err := client.MakeTransformer(ctx, &edge.MakeTransformerRequest{
				Node:       node,
				Javascript: []byte(js),
			})
			if err != nil {
				return errors.Wrap(err, "make transformer")
			}
			stream, err := client.ReadRows(ctx, &edge.ReadRowsRequest{
				Node:          node,
				Start:         []byte{},
				End:           []byte{},
				TransformerId: t.Id,
				Accumulator:   "",
				Limit:         10,
			})
			if err != nil {
				return errors.Wrap(err, "read rows")
			}
			for {
				rows, err := stream.Recv()
				if err != nil {
					return err
				}
				fmt.Printf("%+v\n", rows)
			}
		})
	}
	err = eg.Wait()
	if err != nil {
		logger.Fatal("error executing", zap.Error(err))
	}
}
