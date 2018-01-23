package listeners

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var _ net.Listener = &Reverse{}

// Reverse implements net.Listener but doesn't act like a traditional listener. Instead,
// Reverse dials out to a single target (TODO allow multiple targets) and when this connection
// is establish, makes it available like a listener via Accept(). When the connection is closed
// either remotedly or locally, the connection is re-established and re-provided over Accept().
type Reverse struct {
	cancel func()
	ch     chan net.Conn
	dialer func(context.Context) (net.Conn, error)
}

type ReverseOption func(ctx context.Context, r *Reverse) error

func WithTarget(rawuri string) ReverseOption {
	return func(ctx context.Context, r *Reverse) error {
		u, err := url.Parse(rawuri)
		if err != nil {
			return errors.Wrap(err, "url parse")
		}
		if u.Scheme != "https" {
			return fmt.Errorf("target uri must be https targeturi=%s", rawuri)
		}
		host := u.Host
		addr := u.Host
		if !strings.Contains(host, ":") {
			addr = addr + ":443"
		} else {
			parts := strings.SplitN(host, ":", 2)
			host = parts[0]
		}
		r.dialer = func(ctx context.Context) (net.Conn, error) {
			var conn net.Conn
			d := net.Dialer{}
			operation := func() error {
				zap.L().Info("dialing", zap.String("addr", addr))
				_conn, err := d.DialContext(ctx, "tcp", addr)
				if err != nil {
					return err
				}
				tconn := tls.Client(_conn, &tls.Config{
					ServerName: host,
				})
				// TODO allow timeout
				err = tconn.Handshake()
				if err != nil {
					_conn.Close()
					return errors.Wrap(err, "tls handshake")
				}
				conn = tconn
				return nil
			}
			policy := backoff.NewExponentialBackOff()
			policy.Multiplier = 1.8
			policy.MaxInterval = 5 * time.Minute
			policy.MaxElapsedTime = 0 // allow looping forever until the context is cancelled
			err := backoff.RetryNotify(operation, backoff.WithContext(policy, ctx), func(err error, t time.Duration) {
				zap.L().Info("error while dialing", zap.Error(err))
			})
			if err != nil {
				return nil, err
			}
			return conn, nil
		}
		return nil
	}
}

func NewReverse(ctx context.Context, opts ...ReverseOption) (*Reverse, error) {
	ctx, cancel := context.WithCancel(ctx)
	r := &Reverse{
		cancel: cancel,
		ch:     make(chan net.Conn),
	}
	for _, opt := range opts {
		err := opt(ctx, r)
		if err != nil {
			return nil, err
		}
	}
	go r.run(ctx)
	return r, nil
}

func (r *Reverse) run(ctx context.Context) {
	defer close(r.ch)
	for {
		conn, err := r.dialer(ctx)
		if err != nil {
			return // this closes the channel so that accept will return an error
		}
		done := make(chan struct{})
		conn = &wconn{
			conn: conn,
			onClose: func() {
				close(done)
			},
		}
		select {
		case <-ctx.Done():
			return // this closes the channel so that accept will return an error
		case r.ch <- conn:
		}
		select {
		case <-ctx.Done():
			return
		case <-done:
			// loop to create new connection
		}
	}
}

func (r *Reverse) Accept() (net.Conn, error) {
	conn, active := <-r.ch
	if !active {
		return nil, errors.New("listener has shut down")
	}
	return conn, nil
}

func (r *Reverse) Close() error {
	r.cancel()
	return nil
}

func (r *Reverse) Addr() net.Addr {
	return _addr
}

type addr struct{}

var _addr = &addr{}

func (a *addr) Network() string { return "rgrpc" }
func (a *addr) String() string  { return "" }

type wconn struct {
	conn    net.Conn
	onClose func()
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (wc *wconn) Read(b []byte) (n int, err error) {
	return wc.conn.Read(b)
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (wc *wconn) Write(b []byte) (n int, err error) {
	return wc.conn.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (wc *wconn) Close() error {
	wc.onClose()
	return wc.conn.Close()
}

// LocalAddr returns the local network address.
func (wc *wconn) LocalAddr() net.Addr {
	return wc.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (wc *wconn) RemoteAddr() net.Addr {
	return wc.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (wc *wconn) SetDeadline(t time.Time) error {
	return wc.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (wc *wconn) SetReadDeadline(t time.Time) error {
	return wc.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (wc *wconn) SetWriteDeadline(t time.Time) error {
	return wc.conn.SetWriteDeadline(t)
}
