package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"
	"github.com/supershabam/emitio/eio/pb/edge"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type client struct {
	buckets map[string]int64
	c       edge.EdgeClient
	node    string
	js      string
	l       sync.Mutex
}

func (c *client) sumTo(target map[string]int64) {
	c.l.Lock()
	for key, v := range c.buckets {
		target[key] += v
	}
	c.l.Unlock()
}

func (c *client) run(ctx context.Context) error {
	start := []byte{}
	acc := "{}"
	t, err := c.c.MakeTransformer(ctx, &edge.MakeTransformerRequest{
		Node:       c.node,
		Javascript: []byte(c.js),
	})
	if err != nil {
		return errors.Wrap(err, "make transformer")
	}
Loop:
	stream, err := c.c.Read(ctx, &edge.ReadRequest{
		Node:          c.node,
		Start:         start,
		End:           []byte{},
		TransformerId: t.Id,
		Accumulator:   acc,
		InputLimit:    10000,
		OutputLimit:   9000000,
		DurationLimit: time.Second.Seconds(),
		Tail:          false,
	})
	if err != nil {
		return errors.Wrap(err, "read rows")
	}
	for {
		reply, err := stream.Recv()
		if err != nil {
			if grpc.Code(err) == codes.OutOfRange {
				time.Sleep(time.Second)
				goto Loop
			}
			return err
		}
		next := make([]byte, len(reply.LastInputKey)+1)
		copy(next, reply.LastInputKey)
		next[len(next)-1] = 0x00
		start = next
		var a struct {
			Buckets  map[string]int64 `json:"buckets"`
			Outliers []int            `json:"outliers,omitempty"`
		}
		err = json.Unmarshal([]byte(reply.LastAccumulator), &a)
		if err != nil {
			return errors.Wrap(err, "json unmarshal accumulator")
		}
		if len(a.Outliers) > 0 {
			rows := [][]byte{}
			parts := strings.Split(string(reply.LastInputKey), ":")
			prefix := strings.Join(parts[:len(parts)-1], ":")
			for _, seq := range a.Outliers {
				rows = append(rows, []byte(fmt.Sprintf("%s:%016X", prefix, seq)))
			}
			err = c.outliers(ctx, rows)
			if err != nil {
				return errors.Wrap(err, "while fetching outliers")
			}
		}
		a.Outliers = nil
		b, err := json.Marshal(a)
		if err != nil {
			return errors.Wrap(err, "json marshal accumulator")
		}
		acc = string(b)
		c.l.Lock()
		c.buckets = a.Buckets
		c.l.Unlock()
	}
}

func (c *client) outliers(ctx context.Context, rows [][]byte) error {
	trply, err := c.c.MakeTransformer(ctx, &edge.MakeTransformerRequest{
		Node: c.node,
		Javascript: []byte(`
function transform(acc, line) {
	return [acc, [line]]
}
`),
	})
	if err != nil {
		return errors.Wrap(err, "making outlier transformer")
	}
	eg, ctx := errgroup.WithContext(ctx)
	for _, row := range rows {
		row := row
		eg.Go(func() error {
			stream, err := c.c.Read(ctx, &edge.ReadRequest{
				Start:         row,
				TransformerId: trply.Id,
				InputLimit:    1,
				Node:          c.node,
			})
			if err != nil {
				return errors.Wrap(err, "read stream outlier")
			}
			reply, err := stream.Recv()
			if err != nil {
				if grpc.Code(err) == codes.OutOfRange {
					return nil
				}
				return errors.Wrap(err, "while reading stream")
			}
			fmt.Println(reply.Rows)
			return nil
		})
	}
	return eg.Wait()
}
