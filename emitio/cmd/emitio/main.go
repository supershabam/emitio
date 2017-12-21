package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/supershabam/emitio/emitio/pkg"
	lua "github.com/yuin/gopher-lua"
)

// type ingress struct {
// 	raw         string
// 	fingerprint string
// 	url         *url.URL
// }

// type message struct {
// 	fingerprint string
// 	line        string
// }

// var _ pb.EmitioServer = &server{}

// type server struct {
// 	db *badger.DB
// }

// func (s *server) run(ctx context.Context) error {
// 	eg, ctx := errgroup.WithContext(ctx)
// 	// set up badger
// 	opts := badger.DefaultOptions
// 	dir, err := ioutil.TempDir("", "emitio")
// 	if err != nil {
// 		return err
// 	}
// 	opts.Dir = dir
// 	opts.ValueDir = dir
// 	db, err := badger.Open(opts)
// 	if err != nil {
// 		return err
// 	}
// 	s.db = db
// 	defer db.Close()
// 	// set up output channel
// 	ch := make(chan message)
// 	go func() {
// 		var seq int64
// 		for msg := range ch {
// 			seq = seq + 1
// 			err := db.Update(func(txn *badger.Txn) error {
// 				key := append([]byte(msg.fingerprint), []byte(fmt.Sprintf(":%04x", seq))...)
// 				val := []byte(msg.line)
// 				dur := 5 * time.Minute
// 				return txn.SetWithTTL(key, val, dur)
// 			})
// 			if err != nil {
// 				panic(err)
// 			}
// 		}
// 	}()
// 	// set up listener
// 	l, err := net.Listen("tcp", ":9009")
// 	if err != nil {
// 		return err
// 	}
// 	eg.Go(func() error {
// 		<-ctx.Done()
// 		return errors.Wrap(l.Close(), "listener close")
// 	})
// 	eg.Go(func() error {
// 		for {
// 			conn, err := l.Accept()
// 			if err != nil {
// 				if strings.Contains(err.Error(), "use of closed network connection") {
// 					return nil
// 				}
// 				return err
// 			}
// 			eg.Go(func() error {
// 				<-ctx.Done()
// 				return conn.Close()
// 			})
// 			eg.Go(func() error {
// 				buf := make([]byte, 32*1024)
// 				rbuf := make([]byte, 4*1024)
// 				for {
// 					select {
// 					case <-ctx.Done():
// 						return nil
// 					default:
// 					}
// 					nr, err := conn.Read(rbuf)
// 					if nr > 0 {
// 						buf = append(buf, rbuf[:nr]...)
// 						for {
// 							idx := bytes.IndexByte(buf, '\n')
// 							if idx < 0 {
// 								break
// 							}
// 							line := buf[:idx]
// 							msg := message{
// 								fingerprint: "tcpingress",
// 								line:        string(line),
// 							}
// 							select {
// 							case <-ctx.Done():
// 								return nil
// 							case ch <- msg:
// 							}
// 							buf = buf[idx+1:]
// 						}
// 					}
// 					if err != nil {
// 						if err == io.EOF {
// 							return nil
// 						}
// 						if strings.Contains(err.Error(), "use of closed network connection") {
// 							return nil
// 						}
// 						return err
// 					}
// 				}
// 			})
// 		}
// 	})
// 	go func() {
// 		t := time.NewTicker(time.Second)
// 		defer t.Stop()
// 		for {
// 			select {
// 			case <-t.C:
// 				err := db.View(func(txn *badger.Txn) error {
// 					opts := badger.DefaultIteratorOptions
// 					opts.PrefetchSize = 10
// 					it := txn.NewIterator(opts)
// 					for it.Rewind(); it.Valid(); it.Next() {
// 						item := it.Item()
// 						k := item.Key()
// 						v, err := item.Value()
// 						if err != nil {
// 							return err
// 						}
// 						fmt.Printf("key=%s, value=%s\n", k, v)
// 					}
// 					return nil
// 				})
// 				if err != nil {
// 					panic(err)
// 				}
// 			}
// 		}
// 	}()
// 	err = eg.Wait()
// 	if err != nil {
// 		return err
// 	}
// 	close(ch)
// 	return nil
// }

// func (s *server) ReadRows(ctx context.Context, req *pb.ReadRowsRequest) (*pb.ReadRowsReply, error) {
// 	reply := &pb.ReadRowsReply{
// 		Rows: []*pb.ReadRowsReply_Row{},
// 	}
// 	err := s.db.View(func(txn *badger.Txn) error {
// 		opts := badger.DefaultIteratorOptions
// 		it := txn.NewIterator(opts)
// 		it.Seek(req.Start)
// 		for {
// 			if !it.Valid() {
// 				break
// 			}
// 			item := it.Item()
// 			if len(req.End) > 0 && bytes.Compare(item.Key(), req.End) != -1 {
// 				break
// 			}
// 			value, err := item.Value()
// 			if err != nil {
// 				return err
// 			}
// 			valueCopy := make([]byte, len(value))
// 			copy(valueCopy, value)
// 			reply.Rows = append(reply.Rows, &pb.ReadRowsReply_Row{
// 				Row:   item.Key(),
// 				Value: valueCopy,
// 			})
// 			it.Next()
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return reply, nil
// }

func process(in []byte) ([][]byte, error) {
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
function transform (input)
	return {"one", "two"}
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

	out, err := process([]byte("hi"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("out=%s\n", out)

	ctx := context.TODO()
	c := &pkg.Controller{
		Origin: map[string]string{
			"hostname": "somehost",
		},
	}
	err = c.Run(ctx, []string{
		"syslog+udp://0.0.0.0:9007",
		"syslog+udp://0.0.0.0:9008",
	})
	if err != nil {
		panic(err)
	}
}
