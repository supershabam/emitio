package ingresses

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/supershabam/emitio/emitio/pkg"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var _ pkg.Ingresser = &UDP{}

// UDP produced data by listening for udp packets on the network
type UDP struct {
	uri *url.URL
}

// Ingress implements pkg.Ingresser
func (u *UDP) Ingress(ctx context.Context) (<-chan string, pkg.Wait) {
	ch := make(chan string)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		defer zap.L().Debug("ingress closing", zap.String("uri", u.URI()))
		zap.L().Debug("udp ingress listening", zap.String("addr", u.uri.Host))
		pc, err := net.ListenPacket("udp", u.uri.Host)
		if err != nil {
			return errors.Wrap(err, "listen packet")
		}
		eg.Go(func() error {
			<-ctx.Done()
			return pc.Close()
		})
		buf := make([]byte, 32*1024)
		for {
			nr, _, err := pc.ReadFrom(buf)
			if nr > 0 {
				message := make([]byte, nr)
				copy(message, buf[:nr])
				select {
				case <-ctx.Done():
					return nil
				case ch <- string(message):
					zap.L().Debug("sent message", zap.String("ingress", u.URI()))
					pkg.IncIngressMessage(ctx, u.URI())
				}
			}
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return nil
				}
				return errors.Wrap(err, "read")
			}
		}
	})
	return ch, eg.Wait
}

// URI returns the uri used to instantiate the ingress
func (u *UDP) URI() string {
	return u.uri.String()
}
