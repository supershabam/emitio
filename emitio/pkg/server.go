package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

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
	s            Storage
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
			// key := fmt.Sprintf("%s:%016X", uri, myseq)
			if len(start) == 0 {
				start = []byte("temp:0")
			}
			zap.L().Debug("parsing key for uri and start int", zap.ByteString("start", start))
			parts := strings.Split(string(start), ":")
			uri := strings.Join(parts[:len(parts)-1], ":")
			startInt, err := strconv.ParseInt(parts[len(parts)-1], 16, 64)
			if err != nil {
				return err
			}
			endInt := int64(math.MaxInt64)
			if len(end) > 0 {
				parts = strings.Split(string(end), ":")
				i, err := strconv.ParseInt(parts[len(parts)-1], 16, 64)
				if err != nil {
					return err
				}
				endInt = i
			}
			ch, wait := s.s.Read(ctx, uri, startInt, endInt, maxLen)
			for records := range ch {
				rows := make([]row, 0, len(records))
				for _, record := range records {
					type message struct {
						At       time.Time `json:"a"`
						Raw      string    `json:"r"`
						Sequence int       `json:"s"`
					}
					m := message{
						At:       record.At,
						Raw:      string(record.Blob),
						Sequence: int(record.Seq),
					}
					b, err := json.Marshal(m)
					if err != nil {
						return err
					}
					rows = append(rows, row{
						key:   []byte(fmt.Sprintf("%s:%016X", uri, record.Seq)),
						value: string(b),
					})
				}
				batch = append(batch, rows...)
				flush()
			}
			err = wait()
			if err != nil {
				return err
			}
			flush()
			return nil
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
				lines := []string{}
				var last []byte
				for _, r := range rows {
					lines = append(lines, r.value)
					last = r.key
				}
				acc, out, err := t.Transform(ctx, reply.LastAccumulator, lines)
				if err != nil {
					return errors.Wrap(err, "transform")
				}
				inputCount += len(lines)
				dirty = true
				reply.LastAccumulator = acc
				reply.LastInputKey = last
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
	})
	return ch, eg.Wait
}

// Read queries the embedded key/value database and streams results back to the client. The frequency
// at which responses are sent can be tuned by setting the input_limit, output_limit, and max_duration variables.
func (s *Server) Read(req *emitio.ReadRequest, stream emitio.Emitio_ReadServer) error {
	zap.L().Debug("starting read request")
	ti, ok := s.transformers.Load(req.TransformerId)
	if !ok {
		err := grpc.Errorf(codes.NotFound, "transformer id not found")
		zap.L().Error("loading transformer", zap.Error(err))
		return err
	}
	t, ok := ti.(Transformer)
	if !ok {
		panic(fmt.Sprintf("expected transformer type to be set into map but found %T", ti))
	}
	start := req.Start
	acc := req.Accumulator
Loop:
	s.cond.L.Lock()
	count := s.count
	s.cond.L.Unlock()
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	rowsCh, waitRows := s.batch(ctx, start, req.End, 500)
	replyCh, waitReply := transform(
		ctx,
		t,
		start,
		acc,
		rowsCh,
		int(req.InputLimit),
		int(req.OutputLimit),
		time.Duration(req.DurationLimit*1e9),
	)
	for {
		select {
		case <-ctx.Done():
			return nil
		case reply, active := <-replyCh:
			if !active {
				cancel()
				err := waitReply()
				if err != nil {
					zap.L().Error("wait reply", zap.Error(err))
					return err
				}
				err = waitRows()
				if err != nil {
					zap.L().Error("wait rows", zap.Error(err))
					return err
				}
				if req.Tail {
					s.wait(stream.Context(), count)
					next := make([]byte, len(start)+1)
					copy(next, start)
					next[len(next)-1] = 0x00
					start = next
					goto Loop
				}
				return nil
			}
			err := stream.Send(reply)
			if err != nil {
				zap.L().Error("send", zap.Error(err))
				return err
			}
			start = reply.LastInputKey
			acc = reply.LastAccumulator
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
			seq := 0
			// TODO get seq from storage
			msgch, wait := i.Ingress(ctx)
			for {
				select {
				case <-ctx.Done():
					return nil
				case msg, active := <-msgch:
					if !active {
						return wait()
					}
					seq++
					r := Record{
						At:   time.Now(),
						Blob: []byte(msg),
					}
					err := s.s.Write(ctx, i.URI(), []Record{r})
					if err != nil {
						return err
					}
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

func WithStorage(store Storage) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.s = store
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
