package pkg

import (
	"bytes"
	"context"
	"net"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

var _ Ingresser = &UDP{}

type UDP struct {
	network, addr string
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
				for {
					idx := bytes.IndexByte(buf, ' ')
					if idx == -1 {
						break
					}
					u, err := strconv.ParseUint(string(buf[:idx]), 10, 0)
					if err != nil {
						return err
					}
					msg := Message{
						Origin: map[string]string{
							"ingress": "syslog+udp://TODO",
						},
						What: map[string]interface{}{
							"message": buf[:idx][:u],
						},
					}
					select {
					case <-ctx.Done():
						return nil
					case ch <- msg:
					}
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
