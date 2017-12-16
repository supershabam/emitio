package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type ingress struct {
	raw         string
	fingerprint string
	url         *url.URL
}

type message struct {
	fingerprint string
	line        string
}

type server struct{}

func (s *server) run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	// set up badger
	opts := badger.DefaultOptions
	dir, err := ioutil.TempDir("", "emitio")
	if err != nil {
		return err
	}
	opts.Dir = dir
	opts.ValueDir = dir
	db, err := badger.Open(opts)
	if err != nil {
		return err
	}
	defer db.Close()
	// set up output channel
	ch := make(chan message)
	go func() {
		var seq int64
		for msg := range ch {
			seq = seq + 1
			err := db.Update(func(txn *badger.Txn) error {
				key := append([]byte(msg.fingerprint), []byte(fmt.Sprintf(":%04x", seq))...)
				val := []byte(msg.line)
				dur := 5 * time.Minute
				return txn.SetWithTTL(key, val, dur)
			})
			if err != nil {
				panic(err)
			}
		}
	}()
	// set up listener
	l, err := net.Listen("tcp", ":9009")
	if err != nil {
		return err
	}
	eg.Go(func() error {
		<-ctx.Done()
		return errors.Wrap(l.Close(), "listener close")
	})
	eg.Go(func() error {
		for {
			conn, err := l.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return nil
				}
				return err
			}
			eg.Go(func() error {
				<-ctx.Done()
				return conn.Close()
			})
			eg.Go(func() error {
				buf := make([]byte, 32*1024)
				rbuf := make([]byte, 4*1024)
				for {
					select {
					case <-ctx.Done():
						return nil
					default:
					}
					nr, err := conn.Read(rbuf)
					if nr > 0 {
						buf = append(buf, rbuf[:nr]...)
						for {
							idx := bytes.IndexByte(buf, '\n')
							if idx < 0 {
								break
							}
							line := buf[:idx]
							msg := message{
								fingerprint: "tcpingress",
								line:        string(line),
							}
							select {
							case <-ctx.Done():
								return nil
							case ch <- msg:
							}
							buf = buf[idx+1:]
						}
					}
					if err != nil {
						if err == io.EOF {
							return nil
						}
						if strings.Contains(err.Error(), "use of closed network connection") {
							return nil
						}
						return err
					}
				}
			})
		}
	})
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
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
					panic(err)
				}
			}
		}
	}()
	err = eg.Wait()
	if err != nil {
		return err
	}
	close(ch)
	return nil
}

func main() {
	// --origin pod=$(pod_name)
	// --origin namespace=$(k8s_namespace)
	// --origin datacenter=nyc2
	// --ingress tail:///var/log/message
	// --ingress syslog+udp://0.0.0.0:514/?my_tag=value
	// --ingress ndjson+stdin:///
	// --ingress statsd+udp://0.0.0.0:9001/?region=nyc2#application=something
	// --ingress tail:///var/log/mongodb/mongodb.log#hint=mongodb-v3.18
	// --ingress opentracing+udp://0.0.0.0:9002/
	// --forward https://ingress.emit.io/
	// --listen 0.0.0.0:8080

	/*
		TODO listen on tcp (easier for me to understand) and write each payload into
		the buffer with a "${fingerprint}:${seq}" and TTL of 5m
	*/
	// origin := map[string]string{
	// 	"pod":        "some-pod",
	// 	"namespace":  "production",
	// 	"datacenter": "nyc2",
	// }
	// ingressi := []ingress{
	// 	{
	// 		raw: "tail:///var/log/message",
	// 	}
	// }
	// u, err := url.Parse("mongodb-v3.18+tail:///var/log/mongodb/mongodb.log")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(u.String())

	// set up context
	ctx, cancel := context.WithCancel(context.Background())
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt)
	go func() {
		<-term
		cancel()
		<-term
		os.Exit(1)
	}()
	s := &server{}
	err := s.run(ctx)
	if err != nil {
		panic(err)
	}
}
