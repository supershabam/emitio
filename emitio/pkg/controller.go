package pkg

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

type Controller struct {
	Origin map[string]string
}

func (c *Controller) Run(ctx context.Context, uris []string) error {
	ingresses := make([]Ingresser, 0, len(uris))
	for _, uri := range uris {
		i, err := ParseIngress(uri)
		if err != nil {
			return err
		}
		ingresses = append(ingresses, i)
	}
	eg, ctx := errgroup.WithContext(ctx)
	out := make(chan Message)
	defer close(out)
	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case msg, active := <-out:
				if !active {
					return nil
				}
				// set the controller's origin tags on message but don't override anything
				for k, v := range c.Origin {
					if _, ok := msg.Origin[k]; !ok {
						msg.Origin[k] = v
					}
				}
				fmt.Printf("msg=%+v\n", msg)
			}
		}
	})
	for _, i := range ingresses {
		i := i // capture i locally
		eg.Go(func() error {
			ch, wait := i.Ingress(ctx)
			for {
				select {
				case <-ctx.Done():
					return nil
				case msg, active := <-ch:
					if !active {
						return wait()
					}
					select {
					case <-ctx.Done():
						return nil
					case out <- msg:
					}
				}
			}
		})
	}
	return eg.Wait()
}
