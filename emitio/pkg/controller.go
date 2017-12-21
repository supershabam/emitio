package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type Controller struct {
	Origin map[string]string
}

func (c *Controller) Run(ctx context.Context, uris []string) error {
	opts := badger.DefaultOptions
	dir, err := ioutil.TempDir("", "emitio")
	if err != nil {
		return errors.Wrap(err, "creating temporary directory")
	}
	opts.Dir = dir
	opts.ValueDir = dir
	db, err := badger.Open(opts)
	if err != nil {
		return errors.Wrap(err, "opening badger")
	}
	defer db.Close()
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
	// sink function
	eg.Go(func() error {
		// TODO persist and get the sequence for an ingress upon bootup
		seqs := map[string]uint64{}
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
				// TODO batch these together
				db.Update(func(txn *badger.Txn) error {
					ingress := msg.Origin["ingress"]
					seq := seqs[ingress]
					seqs[ingress]++
					key := []byte(fmt.Sprintf("%s:%08X", ingress, seq))
					val, err := json.Marshal(msg)
					if err != nil {
						return err
					}
					dur := time.Minute
					return txn.SetWithTTL(key, val, dur)
				})
			}
		}
	})
	eg.Go(func() error {
		t := time.NewTicker(time.Second * 10)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				err := db.View(func(txn *badger.Txn) error {
					opts := badger.DefaultIteratorOptions
					opts.PrefetchSize = 10
					it := txn.NewIterator(opts)
					for it.Rewind(); it.Valid(); it.Next() {
						item := it.Item()
						k := item.Key()
						v, err := item.Value()
						if err != nil {
							return err
						}
						fmt.Printf("key=%s, value=%s\n", k, v)
					}
					return nil
				})
				if err != nil {
					return err
				}
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
