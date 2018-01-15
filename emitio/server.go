package emitio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/robertkrimen/otto"
	uuid "github.com/satori/go.uuid"
	"github.com/supershabam/emitio/emitio/pb"
)

var _ pb.EmitioServer = &Server{}

type Server struct {
	db        *badger.DB
	ingresses []Ingresser
}

func NewServer(ctx context.Context, opts ...ServerOption) (*Server, error) {
	s := &Server{
		ingresses: []Ingresser{},
	}
	for _, opt := range opts {
		err := opt(ctx, s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, i := range s.ingresses {
		i := i // capture local variable
		eg.Go(func() error {
			uri := i.URI()
			seq := 0
			// get sequence from database
			s.db.View(func(txn *badger.Txn) error {
				opts := badger.DefaultIteratorOptions
				opts.PrefetchValues = false
				it := txn.NewIterator(opts)
				prefix := []byte(uri)
				var key []byte
				for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
					item := it.Item()
					key = item.Key()
					fmt.Println(key)
				}
				if key != nil {
					parts := bytes.Split(key, []byte{':'})
					i, err := strconv.ParseInt(string(parts[len(parts)-1]), 16, 0)
					if err != nil {
						return err
					}
					seq = int(i)
				}
				return nil
			})
			msgch, wait := i.Ingress(ctx)
			for {
				select {
				case <-ctx.Done():
					return nil
				case msg, active := <-msgch:
					if !active {
						return wait()
					}
					type message struct {
						At       time.Time `json:"a"`
						Raw      string    `json:"r"`
						Sequence int       `json:"s"`
					}
					seq++
					myseq := seq
					b, err := json.Marshal(message{
						At:       time.Now(),
						Raw:      msg,
						Sequence: myseq,
					})
					if err != nil {
						return err
					}
					key := fmt.Sprintf("%s:%016X", uri, myseq)
					s.db.Update(func(txn *badger.Txn) error {
						dur := time.Minute * 30
						return txn.SetWithTTL([]byte(key), b, dur)
					})
					fmt.Printf("%s:%s\n", key, b)
				}
			}
		})
	}
	return eg.Wait()
}

type ServerOption func(context.Context, *Server) error

func WithDB(db *badger.DB) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.db = db
		return nil
	}
}

func WithIngresses(is []Ingresser) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.ingresses = is
		return nil
	}
}

type mockt struct{}

func (m *mockt) Transform(ctx context.Context, acc string, in []string) (string, []string, error) {
	vm := otto.New()
	_, err := vm.Run(`function transform(acc, input) {
	return ['accumulator!', ['bullshit', 'something else']]
}`)
	if err != nil {
		return "", nil, errors.Wrap(err, "compile")
	}
	v, err := vm.Get("transform")
	if err != nil {
		return "", nil, errors.Wrap(err, "get transform")
	}
	rv, err := v.Call(v, acc, in)
	if err != nil {
		return "", nil, err
	}
	result, err := rv.Export()
	if err != nil {
		return "", nil, err
	}
	r, ok := result.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("expected js result to be an array but it is %T", result)
	}
	if len(r) < 2 {
		return "", nil, errors.New("expected js result to be an array of at least len 2")
	}
	acci := r[0]
	outi := r[1]
	acc, ok = acci.(string)
	if !ok {
		return "", nil, errors.New("expected js result first element to be string")
	}
	outii, ok := outi.([]string)
	if !ok {
		return "", nil, fmt.Errorf("expected js result second element to be array but it is %T", outi)
	}
	// out := []string{}
	// for _, i := range outii {
	// 	str, ok := i.(string)
	// 	if !ok {
	// 		return "", nil, errors.New("expected element of out array to be string but wasn't")
	// 	}
	// 	out = append(out, str)
	// }
	return acc, outii, nil
}

func (s *Server) ReadRows(req *pb.ReadRowsRequest, stream pb.Emitio_ReadRowsServer) error {
	t := &mockt{}
	const (
		maxBatchSize = 25
	)
	start := req.Start
	accumulator := req.Accumulator
	count := 0
	for {
		input := []string{}
		err := s.db.View(func(txn *badger.Txn) error {
			last := start
			defer func() {
				start = make([]byte, len(last)+1)
				copy(start, last)
				start[len(start)-1] = '\x00'
			}()
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = maxBatchSize
			it := txn.NewIterator(opts)
			for it.Seek(start); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				if len(req.End) > 0 && bytes.Compare(k, req.End) == -1 {
					return nil
				}
				v, err := item.Value()
				if err != nil {
					return err
				}
				last = k
				input = append(input, string(v))
				count++
				if count >= maxBatchSize {
					return nil
				}
			}
			return nil
		})
		if len(input) > 0 {
			tctx, cancel := context.WithTimeout(stream.Context(), time.Second*10)
			var out []string
			accumulator, out, err = t.Transform(tctx, accumulator, input)
			cancel()
			if err != nil {
				return err
			}
			err = stream.Send(&pb.ReadRowsReply{
				Rows:            out,
				LastInputRow:    start[:len(start)-1],
				LastAccumulator: accumulator,
			})
			if err != nil {
				return err
			}
		}
		time.Sleep(time.Second)
	}
}

func (s *Server) MakeTransformer(ctx context.Context, req *pb.MakeTransformerRequest) (*pb.MakeTransformerReply, error) {
	id := uuid.NewV4().String()
	return &pb.MakeTransformerReply{
		Id: id,
	}, nil
}

func (s *Server) GetIngresses(context.Context, *pb.GetIngressesRequest) (*pb.GetIngressesReply, error) {
	reply := &pb.GetIngressesReply{}
	reply.Ingresses = make([]string, 0, len(s.ingresses))
	for _, i := range s.ingresses {
		reply.Ingresses = append(reply.Ingresses, i.URI())
	}
	return reply, nil
}
