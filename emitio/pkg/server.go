package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/supershabam/emitio/emitio/pb/emitio"
	"github.com/supershabam/emitio/emitio/pkg/transformers"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var _ emitio.EmitioServer = &Server{}

// Server implements the emitio server defined in proto. It allows a remote to query information
// about the running state of the server and also to query information out of the embedded key/value
// store and mutate that information server-side via javascript.
type Server struct {
	db           *badger.DB
	key          string
	id           string
	ingresses    []Ingresser
	origin       map[string]string
	transformers sync.Map

	// cond protects count which is the count of rows written to the server
	cond  sync.Cond
	count uint64
}

// NewServer initializes a server and will error if dependencies are missing.
func NewServer(ctx context.Context, opts ...ServerOption) (*Server, error) {
	s := &Server{
		ingresses: []Ingresser{},
		origin:    map[string]string{},
		cond: sync.Cond{
			L: &sync.Mutex{},
		},
	}
	for _, opt := range opts {
		err := opt(ctx, s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Info returns the API key, id, origin tags, and ingresses that define this running server.
func (s *Server) Info(ctx context.Context, req *emitio.InfoRequest) (*emitio.InfoReply, error) {
	ingresses := []string{}
	for _, i := range s.ingresses {
		ingresses = append(ingresses, i.URI())
	}
	return &emitio.InfoReply{
		Key:       s.key,
		Id:        s.id,
		Origin:    s.origin,
		Ingresses: ingresses,
	}, nil
}

// MakeTransformer accepts a javascript payload that is expected to have a global
// "transform(acc:string, inputs:[string]):(string,[string])" method.
// TODO content hash the javascript and only create a new VM if there isn't one
// already created for the provided javascript. It is possible for the same script to
// be sent to us.
// TODO expire unused javascript VMs
func (s *Server) MakeTransformer(ctx context.Context, req *emitio.MakeTransformerRequest) (*emitio.MakeTransformerReply, error) {
	id := uuid.NewV4().String()
	t, err := transformers.NewJS(string(req.Javascript))
	if err != nil {
		return nil, errors.Wrap(err, "new js")
	}
	s.transformers.Store(id, t)
	return &emitio.MakeTransformerReply{
		Id: id,
	}, nil
}

// ReadRows queries the embedded key/value database and streams results back to the client. The frequency
// at which responses are sent can be tuned by setting the input_limit, output_limit, and max_duration variables.
func (s *Server) ReadRows(req *emitio.ReadRowsRequest, stream emitio.Emitio_ReadRowsServer) error {
	ti, ok := s.transformers.Load(req.TransformerId)
	if !ok {
		return grpc.Errorf(codes.NotFound, "transformer id not found")
	}
	t, ok := ti.(Transformer)
	if !ok {
		panic(fmt.Sprintf("expected transformer type to be set into map but found %T", ti))
	}
	var (
		inputCount  uint32
		outputCount uint32
	)
	const (
		batchSize int = 25
	)
	reply := &emitio.ReadRowsReply{
		Rows:            []string{},
		LastAccumulator: req.Accumulator,
		LastInputRowKey: req.Start,
	}
	for {
		var deadline time.Time
		if req.MaxDuration != 0 {
			deadline = time.Now().Add(time.Duration(req.MaxDuration * 1e9))
		}
		s.cond.L.Lock()
		count := s.count
		s.cond.L.Unlock()
		batch, rerr := s.read(reply.LastInputRowKey, req.End, batchSize)
		if rerr != nil {
			if rerr != io.EOF {
				return rerr
			}
		}
		// clip batch to max allowed by input limit if set
		if req.InputLimit > 0 && inputCount+uint32(len(batch)) >= req.InputLimit {
			batch = batch[:int(req.InputLimit)-len(batch)]
		}
		var outputLimit uint32
		if req.OutputLimit > 0 {
			outputLimit = req.OutputLimit - outputCount
		}
		var rows []string
		var err error
		reply.LastAccumulator, rows, reply.LastInputRowKey, err = s.transform(stream.Context(), t, reply.LastAccumulator, batch, outputLimit)
		if err != nil {
			return err
		}
		reply.Rows = append(reply.Rows, rows...)
		outputCount += uint32(len(rows))
		// TODO add min_output to request
		// decide whether or not to send
		if rerr == io.EOF ||
			(req.InputLimit > 0 && inputCount >= req.InputLimit) ||
			(req.OutputLimit > 0 && outputCount >= req.OutputLimit) {
			err = stream.Send(reply)
			if err != nil {
				return err
			}
			reply = &emitio.ReadRowsReply{
				Rows:            []string{},
				LastInputRowKey: reply.LastInputRowKey,
				LastAccumulator: reply.LastAccumulator,
			}
			inputCount = 0
			outputCount = 0
			// decide whether to end loop
			if rerr == io.EOF {
				return nil
			}
			continue
		}
		// decide on whether or not to wait
		if len(batch) == batchSize {
			continue
		}
		s.wait(stream.Context(), deadline, count)
		// if we're resuming because of context cancellation, we need to send and exit
		done := false
		select {
		case <-stream.Context().Done():
			done = true
		default:
		}
		if done {
			err = stream.Send(reply)
			if err != nil {
				return err
			}
			return nil
		}
		// if we're resuming because of a deadline, we need to send and loop
		if !deadline.IsZero() && time.Now().After(deadline) {
			err = stream.Send(reply)
			if err != nil {
				return err
			}
			reply = &emitio.ReadRowsReply{
				Rows:            []string{},
				LastInputRowKey: reply.LastInputRowKey,
				LastAccumulator: reply.LastAccumulator,
			}
			inputCount = 0
			outputCount = 0
		}
		// otherwise, we just need to loop
	}
}

type row struct {
	key   []byte
	value string
}

func (s *Server) transform(ctx context.Context, t Transformer, acc string, rows []row, limit uint32) (_acc string, _rows []string, last []byte, err error) {
	_rows = []string{}
	for _, r := range rows {
		acc, out, _err := t.Transform(ctx, acc, r.value)
		if _err != nil {
			err = _err
			return
		}
		_acc = acc
		last = r.key
		_rows = append(_rows, out...)
		if limit != 0 && uint32(len(_rows)) >= limit {
			return
		}
	}
	return
}

// wait blocks until any of the following are true
// * ctx closes
// * the deadline (if non-zero) is reached
// * the server's count no longer matches count
func (s *Server) wait(ctx context.Context, deadline time.Time, count uint64) {
	if !deadline.IsZero() {
		_ctx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()
		ctx = _ctx
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		s.cond.Broadcast() // cause condition to be rechecked
	}()
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	for {
		done := false
		select {
		case <-ctx.Done():
			done = true
		default:
		}
		if done || count != s.count {
			return
		}
		s.cond.Wait()
	}
}

func (s *Server) read(start, end []byte, limit int) (batch []row, err error) {
	batch = []row{}
	err = s.db.View(func(txn *badger.Txn) error {
		// the row key []byte is only valid within the txn, so we must allocate
		// and copy data into a new slice to survive exiting the function
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 25
		it := txn.NewIterator(opts)
		for it.Seek(start); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if len(end) > 0 && bytes.Compare(k, end) == -1 {
				return io.EOF
			}
			v, err := item.Value()
			if err != nil {
				return err
			}
			r := row{
				key:   make([]byte, len(k)),
				value: string(v),
			}
			copy(r.key, k)
			batch = append(batch, r)
			if limit > 0 && len(batch) >= limit {
				return nil
			}
		}
		return io.EOF
	})
	return
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
					s.cond.L.Lock()
					s.count++
					s.cond.L.Unlock()
					s.cond.Broadcast()
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

func WithKey(key string) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.key = key
		return nil
	}
}

func WithID(id string) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.id = id
		return nil
	}
}

func WithOrigin(origin map[string]string) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.origin = origin
		return nil
	}
}
