package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/sync/errgroup"
)

type Controller struct {
	Origin map[string]string
}

func (c *Controller) process(in []byte) ([][]byte, error) {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()
	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage}, // Must be first
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
	} {
		if err := L.CallByParam(lua.P{
			Fn:      L.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			return nil, errors.Wrap(err, "loading lua module")
		}
	}
	const program = `
function transform(input)
	return {"one", input }
end
`
	if err := L.DoString(program); err != nil {
		return nil, errors.Wrap(err, "executing lua string")
	}
	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("transform"),
		NRet:    1,
		Protect: true,
	}, lua.LString(in)); err != nil {
		return nil, errors.Wrap(err, "calling transform")
	}
	ret := L.Get(-1) // returned value
	L.Pop(1)         // remove received value
	t, ok := ret.(*lua.LTable)
	if !ok {
		return nil, errors.New("expected return value to be string")
	}
	lines := [][]byte{}
	var outerErr error
	t.ForEach(func(_, v lua.LValue) {
		if outerErr != nil {
			return
		}
		str, ok := v.(lua.LString)
		if !ok {
			outerErr = errors.New("expected table entry to be string")
		}
		lines = append(lines, []byte(str))
	})
	if outerErr != nil {
		return nil, outerErr
	}
	return lines, nil
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
					seqs[ingress]++
					seq := seqs[ingress]
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
	// processing function
	eg.Go(func() error {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		lasts := map[string]uint64{}
		fseqs := map[string]uint64{}
		lua := ""
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				for _, ingress := range ingresses {
					name := ingress.Name()
					last := lasts[name]
					h := fnv.New64a()
					h.Write([]byte(name))
					h.Write([]byte(lua))
					fingerprint := fmt.Sprintf("%0X", h.Sum(nil))
					err := db.Update(func(txn *badger.Txn) error {
						opts := badger.DefaultIteratorOptions
						opts.PrefetchSize = 10
						it := txn.NewIterator(opts)
						start := append([]byte(fmt.Sprintf("%s:%08X", name, last)), 0x00)
						prefix := []byte(fmt.Sprintf("%s:", name))
						for it.Seek(start); it.ValidForPrefix(prefix); it.Next() {
							item := it.Item()
							k := item.Key()
							parts := bytes.Split(k, []byte{':'})
							seq, err := strconv.ParseUint(string(parts[len(parts)-1]), 16, 0)
							if err != nil {
								return err
							}
							lasts[name] = seq
							v, err := item.Value()
							if err != nil {
								return err
							}
							vals, err := c.process(v)
							if err != nil {
								return err
							}
							for _, val := range vals {
								fseqs[string(fingerprint)]++
								fseq := fseqs[string(fingerprint)]
								key := []byte(fmt.Sprintf("%s:%08X", fingerprint, fseq))
								err = txn.SetWithTTL(key, val, time.Minute)
								if err != nil {
									return err
								}
							}
						}
						return nil
					})
					if err != nil {
						return err
					}
				}
			}
		}
	})
	eg.Go(func() error {
		t := time.NewTicker(time.Second * 2)
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
