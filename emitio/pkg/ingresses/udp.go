package ingresses

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/supershabam/emitio/emitio"
	"golang.org/x/sync/errgroup"
)

var _ emitio.Ingresser = &UDP{}

// UDP produced data by listening for udp packets on the network
type UDP struct {
	uri *url.URL
}

// Ingress implements emitio.Ingresser
func (u *UDP) Ingress(ctx context.Context) (<-chan string, emitio.Wait) {
	ch := make(chan string)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
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
