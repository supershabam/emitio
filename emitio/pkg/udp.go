package pkg

import (
	"context"
	"net"
	"strings"

	"golang.org/x/sync/errgroup"
)

var _ Ingresser = &UDP{}

type UDP struct {
	network, addr, uri string
}

func (u *UDP) Ingress(ctx context.Context) (<-chan Message, Wait) {
	ch := make(chan Message)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		pc, err := net.ListenPacket(u.network, u.addr)
		if err != nil {
			return err
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
				msg := Message{
					Origin: map[string]string{
						"ingress": u.uri,
					},
					What: map[string]interface{}{
						"message": string(message),
					},
				}
				select {
				case <-ctx.Done():
					return nil
				case ch <- msg:
				}
			}
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return nil
				}
				return err
			}
		}
	})
	return ch, eg.Wait
}

func (u *UDP) Name() string {
	return u.uri
}
