package pkg

import (
	"context"
	"net"
	"strings"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	uuid "github.com/satori/go.uuid"
	"github.com/supershabam/emitio/edge/pb/edge"
	"github.com/supershabam/emitio/edge/pb/emitio"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var _ edge.EdgeServer = &Server{}

type Server struct {
	nodes map[string]*grpc.ClientConn
	rgrpc *listener
	l     net.Listener
	m     sync.Mutex
}

func NewServer(ctx context.Context, opts ...ServerOption) (*Server, error) {
	s := &Server{
		nodes: map[string]*grpc.ClientConn{},
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
	eg.Go(func() error {
		for {
			cc, wait, err := s.rgrpc.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "listener has shut down") {
					return nil
				}
				return err
			}
			id := uuid.NewV4().String()
			s.m.Lock()
			s.nodes[id] = cc
			s.m.Unlock()
			eg.Go(func() error {
				wait()
				zap.L().Info("removing connection", zap.String("connection_id", id))
				s.m.Lock()
				delete(s.nodes, id)
				s.m.Unlock()
				return nil
			})
		}
	})
	eg.Go(func() error {
		grpcServer := grpc.NewServer()
		edge.RegisterEdgeServer(grpcServer, s)
		err := grpcServer.Serve(s.l)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			return err
		}
		return nil
	})
	eg.Go(func() error {
		<-ctx.Done()
		s.rgrpc.Close()
		s.l.Close()
		return nil
	})
	return eg.Wait()
}

type ServerOption func(ctx context.Context, s *Server) error

func WithRGRPCListener(l net.Listener) ServerOption {
	return func(ctx context.Context, s *Server) error {
		lis := newListener(l)
		s.rgrpc = lis
		return nil
	}
}

func WithAPIListener(l net.Listener) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.l = l
		return nil
	}
}

func (s *Server) Nodes(ctx context.Context, req *edge.NodesRequest) (*edge.NodesReply, error) {
	nodes := []string{}
	s.m.Lock()
	for id := range s.nodes {
		nodes = append(nodes, id)
	}
	s.m.Unlock()
	return &edge.NodesReply{
		Nodes: nodes,
	}, nil
}

func (s *Server) Read(req *edge.ReadRequest, stream edge.Edge_ReadServer) error {
	s.m.Lock()
	cc, ok := s.nodes[req.Node]
	s.m.Unlock()
	if !ok {
		return grpc.Errorf(codes.NotFound, "node not found")
	}
	client := emitio.NewEmitioClient(cc)
	resp, err := client.Read(stream.Context(), &emitio.ReadRequest{
		Start:         req.Start,
		End:           req.End,
		TransformerId: req.TransformerId,
		Accumulator:   req.Accumulator,
		InputLimit:    req.OutputLimit,
		OutputLimit:   req.OutputLimit,
		DurationLimit: req.DurationLimit,
		Tail:          req.Tail,
	})
	if err != nil {
		return err
	}
	for {
		reply, err := resp.Recv()
		if err != nil {
			return err
		}
		err = stream.Send(&edge.ReadReply{
			Rows:            reply.Rows,
			LastAccumulator: reply.LastAccumulator,
			LastInputKey:    reply.LastInputKey,
		})
		if err != nil {
			return err
		}
	}
}

func (s *Server) MakeTransformer(ctx context.Context, req *edge.MakeTransformerRequest) (*edge.MakeTransformerReply, error) {
	s.m.Lock()
	cc, ok := s.nodes[req.Node]
	s.m.Unlock()
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, "node not found")
	}
	client := emitio.NewEmitioClient(cc)
	reply, err := client.MakeTransformer(ctx, &emitio.MakeTransformerRequest{
		Javascript: req.Javascript,
	})
	if err != nil {
		return nil, err
	}
	return &edge.MakeTransformerReply{
		Id: reply.Id,
	}, nil
}

func (s *Server) Info(ctx context.Context, req *edge.InfoRequest) (*edge.InfoReply, error) {
	s.m.Lock()
	cc, ok := s.nodes[req.Node]
	s.m.Unlock()
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, "node not found")
	}
	client := emitio.NewEmitioClient(cc)
	reply, err := client.Info(ctx, &emitio.InfoRequest{})
	if err != nil {
		return nil, err
	}
	return &edge.InfoReply{
		Key:       reply.Key,
		Id:        reply.Id,
		Origin:    reply.Origin,
		Ingresses: reply.Ingresses,
	}, nil
}
