package pkg

import (
	"context"
	"net"
	"strings"
	"sync"

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
	s := &Server{}
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
			cc, err := s.rgrpc.Accept()
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
			// TODO watch client state for changes and remove from map
		}
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

func (s *Server) GetNodes(ctx context.Context, req *edge.GetNodesRequest) (*edge.GetNodesReply, error) {
	nodes := []string{}
	s.m.Lock()
	for id := range s.nodes {
		nodes = append(nodes, id)
	}
	s.m.Unlock()
	return &edge.GetNodesReply{
		Nodes: nodes,
	}, nil
}

func (s *Server) ReadRows(req *edge.ReadRowsRequest, stream edge.Edge_ReadRowsServer) error {
	s.m.Lock()
	cc, ok := s.nodes[req.Node]
	s.m.Unlock()
	if !ok {
		return grpc.Errorf(codes.NotFound, "node not found")
	}
	client := emitio.NewEmitioClient(cc)
	resp, err := client.ReadRows(stream.Context(), &emitio.ReadRowsRequest{
		Start:         req.Start,
		End:           req.End,
		TransformerId: req.TransformerId,
		Accumulator:   req.Accumulator,
		Limit:         req.Limit,
	})
	if err != nil {
		return err
	}
	for {
		reply, err := resp.Recv()
		if err != nil {
			return err
		}
		err = stream.Send(&edge.ReadRowsReply{
			Rows:            reply.Rows,
			LastInputRow:    reply.LastInputRow,
			LastAccumulator: reply.LastAccumulator,
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

func (s *Server) GetIngresses(ctx context.Context, req *edge.GetIngressesRequest) (*edge.GetIngressesReply, error) {
	s.m.Lock()
	cc, ok := s.nodes[req.Node]
	s.m.Unlock()
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, "node not found")
	}
	client := emitio.NewEmitioClient(cc)
	reply, err := client.GetIngresses(ctx, &emitio.GetIngressesRequest{})
	if err != nil {
		return nil, err
	}
	return &edge.GetIngressesReply{
		Ingresses: reply.Ingresses,
	}, nil
}
