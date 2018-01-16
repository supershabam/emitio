package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"
	"github.com/rkt/rkt/tests/testutils/logger"
	"github.com/supershabam/emitio/emitio/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type listener struct {
	cancel func()
	cc     chan *grpc.ClientConn
	ctx    context.Context
	lis    net.Listener
}

func (l *listener) Accept() (net.Conn, error) {
	conn, err := l.lis.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "accept")
	}
	count := 0
	dialer := func(string, time.Duration) (net.Conn, error) {
		count++
		if count > 1 {
			// grpc will attempt to re-establish the connection since it thinks it
			// can dial up a new one. But, since we're doing a reverse-grpc hack, we
			// can't dial the person who dialed us. We have to just fail this redial
			// attempt and expect that the agent will re-dial an edge node. Returning
			// an error allows grpc client to fail.
			return nil, errors.New("dialer called more than once")
		}
		return conn, nil
	}
	cc, err := grpc.DialContext(
		l.ctx,
		"",
		grpc.WithDialer(dialer),
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "dial context")
	}
	select {
	case l.cc <- cc:
	case <-l.ctx.Done():
		return nil, errors.New("context closed")
	}
	return conn, nil
}

func (l *listener) Close() error {
	l.cancel()
	return l.lis.Close()
}

func (l *listener) Addr() net.Addr {
	return l.lis.Addr()
}

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
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	lis := &listener{
		cancel: cancel,
		cc:     make(chan *grpc.ClientConn),
		ctx:    ctx,
		lis:    l,
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		for {
			_, err := lis.Accept()
			if err != nil {
				return err
			}
		}
	})
	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case cc, active := <-lis.cc:
				if !active {
					return nil
				}
				eg.Go(func() error {
					err := process(ctx, cc)
					if err != nil {
						logger.Error("processing client connection", zap.Error(err))
					}
					return nil
				})
			}
		}
	})
	err = eg.Wait()
	time.Sleep(time.Second)
	if err != nil {
		panic(err)
	}
}

func process(ctx context.Context, cc *grpc.ClientConn) error {
	client := pb.NewEmitioClient(cc)
	mtr, err := client.MakeTransformer(ctx, &pb.MakeTransformerRequest{
		Javascript: []byte(`function transform(acc, lines) {
	var a = JSON.parse(acc)
	var output = []
	for (var i = 0; i < lines.length; i++) {
		a.count++
		output.push(lines[i])
	}
	return [JSON.stringify(a), output]
}`),
	})
	if err != nil {
		return err
	}
	stream, err := client.ReadRows(ctx, &pb.ReadRowsRequest{
		TransformerId: mtr.Id,
		Accumulator:   `{"count":0}`,
		Start:         []byte(""),
		End:           []byte(""),
	})
	if err != nil {
		return err
	}
	for {
		reply, err := stream.Recv()
		if err != nil {
			return err
		}
		fmt.Printf("%+v\n", reply)
	}
}
