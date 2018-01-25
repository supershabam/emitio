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

	"go.uber.org/zap"
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

func (s *Server) batch(ctx context.Context, start, end []byte, maxLen int) (chan []row, Wait) {
	ch := make(chan []row)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		batch := make([]row, 0, maxLen)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			select {
			case <-ctx.Done():
			case ch <- batch:
				batch = make([]row, 0, maxLen)
			}
		}
		for {
			// escape hatch for when context cancels
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			err := s.db.View(func(txn *badger.Txn) error {
				// last points to the last read row key value but is only valid within
				// the txn.
				var last []byte
				defer func() {
					start = make([]byte, len(last)+1)
					copy(start, last)
					// start the next iteration at the next possible key
					start[len(start)-1] = 0x00
				}()
				opts := badger.DefaultIteratorOptions
				opts.PrefetchSize = 25
				it := txn.NewIterator(opts)
				for it.Seek(start); it.Valid(); it.Next() {
					item := it.Item()
					k := item.Key()
					// if there is an end AND k >= end
					if len(end) > 0 && bytes.Compare(k, end) >= 0 {
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
					last = r.key
					batch = append(batch, r)
					if len(batch) >= maxLen {
						return nil
					}
				}
				return io.EOF
			})
			if err != nil && err != io.EOF {
				return errors.Wrap(err, "batch db view")
			}
			flush()
			if err == io.EOF {
				return nil
			}
		}
	})
	return ch, eg.Wait
}

func transform(
	ctx context.Context,
	t Transformer,
	last []byte,
	acc string,
	rowsCh <-chan []row,
	maxInput, maxOutput int,
	maxDelay time.Duration,
) (chan *emitio.ReadReply, Wait) {
	ch := make(chan *emitio.ReadReply)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		inputCount := 0
		reply := &emitio.ReadReply{
			Rows:            []string{},
			LastAccumulator: acc,
			LastInputKey:    last,
		}
		dirty := false
		timer := time.NewTimer(maxDelay)
		flush := func() {
			timer.Reset(maxDelay)
			if !dirty {
				return
			}
			select {
			case <-ctx.Done():
			case ch <- reply:
				inputCount = 0
				dirty = false
				reply = &emitio.ReadReply{
					Rows:            []string{},
					LastAccumulator: reply.LastAccumulator,
					LastInputKey:    reply.LastInputKey,
				}
			}
		}
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				flush()
			case rows, active := <-rowsCh:
				if !active {
					flush()
					return nil
				}
				for _, r := range rows {
					acc, out, err := t.Transform(ctx, reply.LastAccumulator, r.value)
					if err != nil {
						return errors.Wrap(err, "transform")
					}
					dirty = true
					inputCount++
					reply.LastAccumulator = acc
					reply.LastInputKey = r.key
					reply.Rows = append(reply.Rows, out...)
					if inputCount >= maxInput || len(reply.Rows) >= maxOutput {
						flush()
						select {
						case <-ctx.Done():
							return nil
						default:
						}
					}
				}
			}
		}
	})
	return ch, eg.Wait
}

// Read queries the embedded key/value database and streams results back to the client. The frequency
// at which responses are sent can be tuned by setting the input_limit, output_limit, and max_duration variables.
func (s *Server) Read(req *emitio.ReadRequest, stream emitio.Emitio_ReadServer) error {
	ti, ok := s.transformers.Load(req.TransformerId)
	if !ok {
		return grpc.Errorf(codes.NotFound, "transformer id not found")
	}
	t, ok := ti.(Transformer)
	if !ok {
		panic(fmt.Sprintf("expected transformer type to be set into map but found %T", ti))
	}
Loop:
	s.cond.L.Lock()
	count := s.count
	s.cond.L.Unlock()
	rowsCh, _ := s.batch(stream.Context(), req.Start, req.End, 25)
	replyCh, _ := transform(
		stream.Context(),
		t,
		req.Start,
		req.Accumulator,
		rowsCh,
		int(req.InputLimit),
		int(req.OutputLimit),
		time.Duration(req.DurationLimit*1e9),
	)
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case reply, active := <-replyCh:
			if !active {
				zap.L().Debug("replyCh has terminated")
				if req.Tail {
					s.wait(stream.Context(), count)
					goto Loop
				}
				return nil
			}
			zap.L().Debug("received reply from replyCh")
			err := stream.Send(reply)
			if err != nil {
				return err
			}
		}
	}
}

type row struct {
	key   []byte
	value string
}

// wait blocks until any of the following are true
// * ctx closes
// * the server's count no longer matches count
func (s *Server) wait(ctx context.Context, count uint64) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		s.cond.Broadcast() // cause condition to be rechecked so function's blocking routine may exit
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
