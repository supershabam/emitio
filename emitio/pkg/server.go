package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/supershabam/emitio/emitio/pb/emitio"
	"github.com/supershabam/emitio/emitio/pkg/transformers"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var _ emitio.EmitioServer = &Server{}

// Server implements the emitio server defined in proto. It allows a remote to query information
// about the running state of the server and also to query information out of the embedded key/value
// store and mutate that information server-side via javascript.
type Server struct {
	s            Storager
	key          string
	id           string
	ingresses    []Ingresser
	origin       map[string]string
	transformers map[string]Transformer
	l            sync.Mutex
}

// NewServer initializes a server and will error if dependencies are missing.
func NewServer(ctx context.Context, opts ...ServerOption) (*Server, error) {
	s := &Server{
		ingresses:    []Ingresser{},
		origin:       map[string]string{},
		transformers: map[string]Transformer{},
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
	s.l.Lock()
	s.transformers[id] = t
	s.l.Unlock()
	return &emitio.MakeTransformerReply{
		Id: id,
	}, nil
}

// Read queries the embedded key/value database and streams results back to the client. The frequency
// at which responses are sent can be tuned by setting the input_limit, output_limit, and max_duration variables.
func (s *Server) Read(req *emitio.ReadRequest, stream emitio.Emitio_ReadServer) error {
	const (
		batchSize = 256
	)
	ctx := stream.Context()
	s.l.Lock()
	t, ok := s.transformers[req.TransformerId]
	s.l.Unlock()
	if !ok {
		return fmt.Errorf("unable to load transformer id=%s", req.TransformerId)
	}
	start := req.Start
	acc := req.Accumulator
	out := []string{}
	flush := func() error {
		if len(out) == 0 {
			return nil
		}
		err := stream.Send(&emitio.ReadReply{
			Rows:            out,
			LastAccumulator: acc,
			LastOffset:      start - 1,
		})
		if err != nil {
			return err
		}
		out = []string{}
		return nil
	}
	// TODO add back limiting controls for flushing
	for {
		ch, wait := s.s.Read(ctx, req.Uri, start, req.End, batchSize)
		for batch := range ch {
			zap.L().Debug("Hi")
			if len(batch) == 0 {
				continue
			}
			start = batch[len(batch)-1].Seq + 1
			input := make([]string, len(batch))
			for _, r := range batch {
				b, err := json.Marshal(map[string]interface{}{
					"s": r.Seq,
					"r": string(r.Blob),
					"a": float64(r.At.UnixNano()) / float64(time.Second),
				})
				if err != nil {
					return err
				}
				input = append(input, string(b))
			}
			a, o, err := t.Transform(ctx, acc, input)
			if err != nil {
				return err
			}
			acc = a
			out = append(out, o...)
			err = flush()
			if err != nil {
				return err
			}
		}
		err := wait()
		if err != nil {
			return err
		}
		if !req.Tail {
			return nil
		}
		err = s.s.Watch(ctx, req.Uri, start-1)
		if err != nil {
			return err
		}
	}
}

func (s *Server) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, i := range s.ingresses {
		i := i // capture local variable
		eg.Go(func() error {
			msgch, wait := i.Ingress(ctx)
			for {
				select {
				case <-ctx.Done():
					return nil
				case msg, active := <-msgch:
					if !active {
						return wait()
					}
					r := Record{
						At:   time.Now(),
						Blob: []byte(msg),
					}
					err := s.s.Write(ctx, i.URI(), []Record{r})
					if err != nil {
						return err
					}
				}
			}
		})
	}
	return eg.Wait()
}

type ServerOption func(context.Context, *Server) error

func WithStorage(store Storager) ServerOption {
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
