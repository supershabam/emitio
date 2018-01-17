package pkg

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type listener struct {
	cancel func()
	ch     chan *grpc.ClientConn
	l      net.Listener
}

func newListener(l net.Listener) *listener {
	ctx, cancel := context.WithCancel(context.Background())
	lis := &listener{
		cancel: cancel,
		ch:     make(chan *grpc.ClientConn),
		l:      l,
	}
	go lis.run(ctx)
	return lis
}

func (l *listener) Accept() (*grpc.ClientConn, error) {
	cc, active := <-l.ch
	if !active {
		return nil, errors.New("listener has shut down")
	}
	return cc, nil
}

func (l *listener) Close() error {
	l.cancel()
	return nil
}

func (l *listener) run(ctx context.Context) {
	defer close(l.ch)
	defer l.l.Close()
	type msg struct {
		conn net.Conn
		err  error
	}
	for {
		next := make(chan msg, 1)
		go func() {
			conn, err := l.l.Accept()
			next <- msg{conn, err}
		}()
		select {
		case <-ctx.Done():
			return
		case msg := <-next:
			if msg.err != nil {
				return
			}
			cc, err := l.cc(ctx, msg.conn)
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case l.ch <- cc:
			}
		}
	}
}

func (l *listener) cc(ctx context.Context, conn net.Conn) (*grpc.ClientConn, error) {
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
	return grpc.DialContext(
		ctx,
		"",
		grpc.WithDialer(dialer),
		grpc.WithInsecure(),
	)
}
