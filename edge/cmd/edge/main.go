package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"
	"github.com/rkt/rkt/tests/testutils/logger"
	"github.com/supershabam/emitio/emitio/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

/**
// A Listener is a generic network listener for stream-oriented protocols.
//
// Multiple goroutines may invoke methods on a Listener simultaneously.
type Listener interface {
	// Accept waits for and returns the next connection to the listener.
	Accept() (Conn, error)

	// Close closes the listener.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Addr returns the listener's network address.
	Addr() Addr
}

// Addr represents a network end point address.
//
// The two methods Network and String conventionally return strings
// that can be passed as the arguments to Dial, but the exact form
// and meaning of the strings is up to the implementation.
type Addr interface {
	Network() string // name of the network (for example, "tcp", "udp")
	String() string  // string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
}
*/

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
	dialer := func(string, time.Duration) (net.Conn, error) {
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
					err := process(cc)
					if err != nil {
						logger.Error("processing client connection", zap.Error(err))
					}
					return nil
				})
			}
		}
	})
	err = eg.Wait()
	if err != nil {
		panic(err)
	}
}

func process(cc *grpc.ClientConn) error {
	ctx := context.TODO()
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
