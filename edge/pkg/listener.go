package pkg

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type waitFn func() error

type listener struct {
	cancel func()
	ch     chan accept
	l      net.Listener
}

type accept struct {
	cc   *grpc.ClientConn
	wait waitFn
}

func newListener(l net.Listener) *listener {
	ctx, cancel := context.WithCancel(context.Background())
	lis := &listener{
		cancel: cancel,
		ch:     make(chan accept),
		l:      l,
	}
	go lis.run(ctx)
	return lis
}

func (l *listener) Accept() (*grpc.ClientConn, waitFn, error) {
	msg, active := <-l.ch
	if !active {
		return nil, nil, errors.New("listener has shut down")
	}
	return msg.cc, msg.wait, nil
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
			cc, wait, err := l.cc(ctx, msg.conn)
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case l.ch <- accept{cc: cc, wait: wait}:
			}
		}
	}
}

func (l *listener) cc(ctx context.Context, conn net.Conn) (*grpc.ClientConn, waitFn, error) {
	var count uint64
	done := make(chan struct{})
	var cc *grpc.ClientConn
	dialer := func(string, time.Duration) (net.Conn, error) {
		zap.L().Info("dialing")
		c := atomic.AddUint64(&count, 1)
		if c == 1 {
			zap.L().Info("opening connection")
			return conn, nil
		}
		if c == 2 {
			zap.L().Info("closing connection")
			cc.Close()
			close(done)
		}
		return nil, errors.New("dialer called more than once")
	}
	cc, err := grpc.DialContext(
		ctx,
		"",
		grpc.WithDialer(dialer),
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "grpc dial context")
	}
	waitFn := func() error {
		<-done
		return nil
	}
	return cc, waitFn, nil
}
