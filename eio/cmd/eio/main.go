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
function transform(acc, line) {
	return [acc, [line]]
}
`

// const js = `
// function transform(acc, line) {
// 	var output = []
// 	var l = JSON.parse(line)
// 	var r = l.r
// 	if (r.indexOf('processor') == -1) {
// 		continue
// 	}
// 	var parts = r.split('\n')
// 	for (var j=0; j<parts.length; j++) {
// 		output.push(parts[j])
// 	}
// 	return [acc, output]
// }
// `

// const js = `
// function transform(acc, lines) {
// 	var output = []
// 	for (var i = 0; i < lines.length; i++) {
// 		var line = lines[i]
// 		if (line.indexOf('sshd') == -1) {
// 			continue
// 		}
// 		if (line.indexOf('Invalid user') == -1) {
// 			continue
// 		}
// 		output.push(line)
// 	}
// 	return [acc, output]
// }`

// const js = `
// function transform(acc, lines) {
// 	var a = JSON.parse(acc)
// 	a.users = (a.users || {})
// 	for (var i = 0; i < lines.length; i++) {
// 		var line = lines[i]
// 		if (line.indexOf('sshd') == -1) {
// 			continue
// 		}
// 		var invalidUserIdx = line.indexOf('Invalid user')
// 		if (invalidUserIdx == -1) {
// 			continue
// 		}
// 		a["count"] = (a["count"] || 0) + 1
// 		var invalidUserExtra = line.substr(invalidUserIdx + 'Invalid user'.length + 1, 10)
// 		var parts = invalidUserExtra.split(' ')
// 		var invalidUser = parts[0]
// 		a.users[invalidUser] = (a.users[invalidUser] || 0) + 1
// 	}
// 	return [JSON.stringify(a), []]
// }
// `

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
	logger.Info("got nodes", zap.Int("node_count", len(reply.Nodes)))
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
				Accumulator:   "{}",
				InputLimit:    20,
				OutputLimit:   10,
			})
			if err != nil {
				return errors.Wrap(err, "read rows")
			}
			acc := ""
			for {
				rows, err := stream.Recv()
				if err != nil {
					return err
				}
				if rows.LastAccumulator != acc {
					acc = rows.LastAccumulator
					fmt.Println(acc)
				}
				if len(rows.Rows) > 0 {
					fmt.Println(rows.Rows)
				}
			}
		})
	}
	err = eg.Wait()
	if err != nil {
		logger.Fatal("error executing", zap.Error(err))
	}
}
