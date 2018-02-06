package pkg

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
	http  net.Listener
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

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	onErr := func(err error) {
		zap.L().Error("serving http", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		return
	}
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		onErr(err)
		return
	}
	defer conn.Close()
	var fn struct {
		Javascript string `json:"javascript"`
	}
	err = conn.ReadJSON(&fn)
	if err != nil {
		zap.L().Error("decode json", zap.Error(err))
		return
	}
	cc, err := grpc.DialContext(r.Context(), s.l.Addr().String(), grpc.WithInsecure())
	if err != nil {
		zap.L().Error("grpc dial", zap.Error(err))
		return
	}
	c := edge.NewEdgeClient(cc)
	nodes, err := c.Nodes(r.Context(), &edge.NodesRequest{})
	if err != nil {
		zap.L().Error("get nodes", zap.Error(err))
		return
	}
	type msg struct {
		node  string
		reply *edge.ReadReply
	}
	ch := make(chan msg)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes.Nodes {
		node := node
		eg.Go(func() error {
			zap.L().Debug("starting request client", zap.String("node", node))
			t, err := c.MakeTransformer(ctx, &edge.MakeTransformerRequest{
				Node:       node,
				Javascript: []byte(fn.Javascript),
			})
			if err != nil {
				return err
			}
			s, err := c.Read(ctx, &edge.ReadRequest{
				TransformerId: t.Id,
				InputLimit:    1000,
				OutputLimit:   1000,
				DurationLimit: time.Second.Seconds(),
				Node:          node,
			})
			if err != nil {
				return err
			}
			for {
				reply, err := s.Recv()
				if err != nil && grpc.Code(err) == codes.OutOfRange {
					return nil
				}
				if err != nil {
					return err
				}
				select {
				case <-ctx.Done():
					return nil
				case ch <- msg{
					node:  node,
					reply: reply,
				}:
				}
			}
		})
	}
	eg.Go(func() error {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return nil
			case m, active := <-ch:
				if !active {
					return nil
				}
				err := conn.WriteJSON(map[string]interface{}{
					"node":             m.node,
					"last_accumulator": m.reply.LastAccumulator,
					"rows":             m.reply.Rows,
					"last_input_key":   string(m.reply.LastInputKey),
				})
				if err != nil {
					return err
				}
			}
		}
	})
	err = eg.Wait()
	if err != nil {
		zap.L().Error("from wait", zap.Error(err))
		return
	}
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
		if s.http == nil {
			return nil
		}
		srv := http.Server{Handler: s}
		return srv.Serve(s.http)
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

func WithHTTPListener(l net.Listener) ServerOption {
	return func(ctx context.Context, s *Server) error {
		s.http = l
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
